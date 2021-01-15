package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/fatih/color"
	"io/ioutil"
	"os"
)

func getCertificates(loc string, remoteChain, remoteInters bool) ([]*x509.Certificate, bool, error) {
	var addr, file string
	var err error
	var certs []*x509.Certificate
	var remote bool

	result, err := parseURL(loc)
	if err == nil {
		addr = result
	} else {
		_, err := os.Stat(loc)
		if !(remoteChain || remoteInters) && err == nil {
			file = loc
		} else {
			result, err = parseURL("certmin://" + loc)
			if err != nil {
				return nil, false, err
			}
			addr = result
		}
	}

	if file != "" {
		certs, err = splitMultiCertFile(file)
		if err != nil {
			return nil, false, err
		}
	} else {
		certs, err = retrieveCerts(addr)
		if err != nil {
			return nil, true, err
		}
		certs = orderRemoteChain(certs)
		remote = true
	}

	switch {
	case !remote:
		return certs, remote, nil
	case !(remoteChain || remoteInters):
		return []*x509.Certificate{certs[0]}, remote, nil
	case remoteChain:
		return certs, remote, nil
	case remoteInters:
		var filtered []*x509.Certificate
		for _, cert := range certs {
			if !isRootCA(cert) {
				filtered = append(filtered, cert)
			}
		}
		return filtered, remote, nil
	default:
		panic("unexpected combination")
	}
}

func isRootCA(cert *x509.Certificate) bool {
	return cert.Subject.String() == cert.Issuer.String()
}

// Just try to order the results and return the original array if
// something fishy is going on
func orderRemoteChain(certs []*x509.Certificate) []*x509.Certificate {
	var ordered []*x509.Certificate
	parentName := make(map[string]string)
	certByName := make(map[string]*x509.Certificate)

	// Get the information needed to follow the chain
	for _, cert := range certs {
		// the chain is fishy
		if _, ok := certByName[cert.Subject.String()]; ok {
			return certs
		}
		if _, ok := parentName[cert.Subject.String()]; ok {
			return certs
		}

		certByName[cert.Subject.String()] = cert
		parentName[cert.Subject.String()] = cert.Issuer.String()
	}

	seen := make(map[string]bool)
	for _, cert := range certs {
		if _, ok := seen[cert.Subject.String()]; ok {
			continue
		}
		ordered = append(ordered, cert)
		for { // follow the chain
			_, ok := certByName[parentName[cert.Subject.String()]] // we have that cert
			_, ok2 := seen[parentName[cert.Subject.String()]]      // the parent has not been seen
			if ok && !ok2 {
				// do we have the next Issuer (e.g. incomplete chain
				if _, ok := certByName[parentName[cert.Subject.String()]]; ok {
					ordered = append(ordered, certByName[parentName[cert.Subject.String()]])
					seen[parentName[cert.Subject.String()]] = true
					cert = certByName[parentName[cert.Subject.String()]]
					continue
				}
			}
			break
		}
	}

	return ordered
}

func splitMultiCertFile(certFile string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	pemData, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
		pemData = rest
	}

	if len(certs) == 0 {
		return nil, errors.New("no certificates found")
	}

	return certs, nil
}

func verifyChainFromX509(roots, inters []*x509.Certificate, cert *x509.Certificate) (bool, string) {
	rootPool := x509.NewCertPool()
	for _, root := range roots {
		rootPool.AddCert(root)
	}

	interPool := x509.NewCertPool()
	for _, inter := range inters {
		interPool.AddCert(inter)
	}

	var verifyOptions x509.VerifyOptions
	if len(rootPool.Subjects()) != 0 {
		verifyOptions.Roots = rootPool
	}
	if len(interPool.Subjects()) != 0 {
		verifyOptions.Intermediates = interPool
	}

	if _, err := cert.Verify(verifyOptions); err != nil {
		return false, color.RedString(err.Error() + "\n")
	}

	return true, ""
}
