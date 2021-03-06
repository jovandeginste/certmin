package certmin

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"time"
)

// RetrieveCertsFromAddr retrieves all the certificates offered by the remote host. As parameters
// it takes an address string in the form of hostname:port and a time-out duration for the
// connection. The time-out is used for both the TCP and the SSL connection, with 0 disabling it.
// The return values are a []*x509.Certificate (with the first element being the certificate
// of the server), an error with a warning (e.g. mismatch between the hostname and the CN or DNS alias
// in the certificate) and an error in case of failure.
func RetrieveCertsFromAddr(addr string, timeOut time.Duration) ([]*x509.Certificate, error, error) {
	conn, err := net.DialTimeout("tcp", addr, timeOut)
	if err != nil {
		return nil, nil, err
	}

	var warning error
	rx := regexp.MustCompile(":\\d+$")
	tlsConn := tls.Client(conn, &tls.Config{ServerName: rx.ReplaceAllString(addr, "")})
	err = tlsConn.SetDeadline(time.Now().Add(timeOut))
	if err != nil {
		return nil, nil, err
	}
	err = tlsConn.Handshake()
	if err != nil {
		tlsConn = tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		err2 := tlsConn.SetDeadline(time.Now().Add(timeOut))
		if err2 != nil {
			return nil, nil, err2
		}
		err2 = tlsConn.Handshake()
		if err2 != nil {
			return nil, nil, err2
		}
		warning = err
	}
	defer tlsConn.Close()
	defer conn.Close()

	if len(tlsConn.ConnectionState().PeerCertificates) == 0 {
		err := errors.New("no certificates found")
		return nil, warning, err
	}

	return tlsConn.ConnectionState().PeerCertificates, warning, nil
}

// RetrieveChainFromIssuerURLs retrieves the chain for a certificate by following the
// Issuing Certificate URLs field in the certificate (if present) and consecutively
// following the Issuing Certificate URLs from issuing certificates. As parameters
// it takes a *x509.Certificate and a time-out duration for the HTTP connection with
// 0 disabling it. The return values are a []*x509.Certificate (with the first element
// being the supplied certificate) and an error in case of failure.
func RetrieveChainFromIssuerURLs(cert *x509.Certificate, timeOut time.Duration) ([]*x509.Certificate, error) {
	var chain []*x509.Certificate
	var lastErr error
	recursiveHopCerts(cert, &chain, &lastErr, timeOut)
	return chain, lastErr
}

// recursiveHopCerts follows the URL links recursively
func recursiveHopCerts(
	cert *x509.Certificate, chain *[]*x509.Certificate, lastErr *error, timeOut time.Duration) *x509.Certificate {
	if cert == nil {
		return nil
	}

	client := http.Client{Timeout: timeOut}
	*chain = append(*chain, cert)
	for _, url := range cert.IssuingCertificateURL {
		resp, err := client.Get(url)
		if err != nil {
			*lastErr = err
			continue
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			*lastErr = err
			continue
		}
		defer resp.Body.Close()

		decodedCerts, err := DecodeCertBytes(bodyBytes, "")
		if err != nil {
			*lastErr = err
			continue
		}

		*lastErr = nil
		return recursiveHopCerts(decodedCerts[0], chain, lastErr, timeOut)
	}

	return nil
}
