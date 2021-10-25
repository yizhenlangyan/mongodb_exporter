package shared

import (
	"time"

	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	"github.com/globalsign/mgo"
	"github.com/golang/glog"
)

const (
	dialMongodbTimeout = 10 * time.Second
	syncMongodbTimeout = 1 * time.Minute
)

// MongoSessionOpts represents options for a Mongo session
type MongoSessionOpts struct {
	URI                   string
	TLSCertificateFile    string
	TLSPrivateKeyFile     string
	TLSCaFile             string
	TLSHostnameValidation bool
	TLSAuth               bool
	UserName              string
	AuthMechanism         string
	SocketTimeout         time.Duration
}

// MongoSession creates a Mongo session
func MongoSession(opts MongoSessionOpts) *mgo.Session {
	dialInfo, err := mgo.ParseURL(opts.URI)
	if err != nil {
		glog.Errorf("Cannot connect to server using url %s: %s", opts.URI, err)
		return nil
	}

	dialInfo.Direct = true // Force direct connection
	dialInfo.Timeout = dialMongodbTimeout
	if opts.UserName != "" {
		dialInfo.Username = opts.UserName
	}

	cred, err := opts.configureDialInfoIfRequired(dialInfo)
	if err != nil {
		glog.Errorf("%s", err)
		return nil
	}

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		glog.Errorf("Cannot connect to server using url %s: %s", opts.URI, err)
		return nil
	}

	// For direct connection, we need to set mode to be Monotonic (similar to SecondaryPreferred) or
	// Eventual (similar to Nearest) before login. Otherwise, mongo exporter can not log in secondary mongod.
	session.SetMode(mgo.Eventual, true)

	if cred != nil {
		if err := session.Login(cred); err != nil {
			glog.Errorf("Cannot login to server using TLS credential: %s", err)
			return nil
		}
	}

	session.SetSyncTimeout(syncMongodbTimeout)
	session.SetSocketTimeout(opts.SocketTimeout)
	return session
}

func (opts MongoSessionOpts) configureDialInfoIfRequired(dialInfo *mgo.DialInfo) (*mgo.Credential, error) {
	if opts.AuthMechanism != "" {
		dialInfo.Mechanism = opts.AuthMechanism
	}
	if len(opts.TLSCertificateFile) > 0 {
		certificate, err := LoadKeyPairFrom(opts.TLSCertificateFile, opts.TLSPrivateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Cannot load key pair from '%s' and '%s' to connect to server '%s'. Got: %v", opts.TLSCertificateFile, opts.TLSPrivateKeyFile, opts.URI, err)
		}
		config := &tls.Config{
			Certificates:       []tls.Certificate{certificate},
			InsecureSkipVerify: !opts.TLSHostnameValidation,
		}
		if len(opts.TLSCaFile) > 0 {
			ca, err := LoadCertificatesFrom(opts.TLSCaFile)
			if err != nil {
				return nil, fmt.Errorf("Couldn't load client CAs from %s. Got: %s", opts.TLSCaFile, err)
			}
			config.RootCAs = ca
		}
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), config)
			if err != nil {
				glog.Infof("Could not connect to %v. Got: %v", addr, err)
				return nil, err
			}
			if config.InsecureSkipVerify {
				err = enrichWithOwnChecks(conn, config)
				if err != nil {
					glog.Infof("Could not disable hostname validation. Got: %v", err)
				}
			}
			return conn, err
		}

		if opts.TLSAuth {
			// Authenticate using the certificate
			c, err := x509.ParseCertificate(certificate.Certificate[0]) // TODO: what if multiple
			if err != nil {
				return nil, err
			}
			return &mgo.Credential{Certificate: c, Source: "$external"}, nil
		}
	}
	return nil, nil
}

func enrichWithOwnChecks(conn *tls.Conn, tlsConfig *tls.Config) error {
	var err error
	if err = conn.Handshake(); err != nil {
		conn.Close()
		return err
	}

	opts := x509.VerifyOptions{
		Roots:         tlsConfig.RootCAs,
		CurrentTime:   time.Now(),
		DNSName:       "",
		Intermediates: x509.NewCertPool(),
	}

	certs := conn.ConnectionState().PeerCertificates
	for i, cert := range certs {
		if i == 0 {
			continue
		}
		opts.Intermediates.AddCert(cert)
	}

	_, err = certs[0].Verify(opts)
	if err != nil {
		conn.Close()
		return err
	}

	return nil
}
