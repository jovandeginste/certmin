package certmin

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/youmark/pkcs8"
	"go.mozilla.org/pkcs7"
	"software.sslmate.com/src/go-pkcs12"
)

// CertTree represents a chain where certificates are
// assigned as a Certificate, Intermediates and Roots.
type CertTree struct {
	Certificate          *x509.Certificate
	Intermediates, Roots []*x509.Certificate
}

// DecodeCertBytes reads a []byte with DER or PEM PKCS1, PKCS7 and PKCS12 encoded certificates,
// and returns the contents as a []*x509.Certificate and an error if encountered. A password is
// only needed for PKCS12.
func DecodeCertBytes(certBytes []byte, password string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	var err error
	var errStrs []string

	for {
		certs, err = DecodeCertBytesPKCS1PEM(certBytes)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		certs, err = DecodeCertBytesPKCS1DER(certBytes)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		certs, err = DecodeCertBytesPKCS7PEM(certBytes)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		certs, err = DecodeCertBytesPKCS7DER(certBytes)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		certs, err = DecodeCertBytesPKCS12(certBytes, password)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		break
	}

	if err != nil {
		return nil, errors.New(strings.Join(errStrs, "   >>   "))
	}

	if len(certs) == 0 {
		return nil, errors.New("no certificates found")
	}

	return certs, nil
}

// DecodeCertBytesPKCS1DER reads a []byte with PKCS1 DER encoded certificates (e.g. read
// from a file of a HTTP response body), and returns the contents as a  []*x509.Certificate
// and an error if encountered. If you don't know in what format the data is encoded, use
// DecodeCertBytes.
func DecodeCertBytesPKCS1DER(certBytes []byte) ([]*x509.Certificate, error) {
	certs, err := x509.ParseCertificates(certBytes)
	if err != nil {
		return nil, err
	}

	if len(certs) == 0 {
		err = errors.New("no certificates found")
	}

	return certs, err
}

// DecodeCertBytesPKCS1PEM reads a []byte with PKCS1 PEM encoded certificates (e.g. read
// from a file of a HTTP response body), and returns the contents as a []*x509.Certificate
// and an error if encountered. If you don't know in what format the data is encoded, use
// DecodeCertBytes.
func DecodeCertBytesPKCS1PEM(certBytes []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	pemBytes := certBytes
	for {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if bytes.Equal(rest, pemBytes) {
			return nil, errors.New("not valid PKCS1 PEM data")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
		pemBytes = rest
	}

	var err error
	if len(certs) == 0 {
		err = errors.New("no certificates found")
	}

	return certs, err
}

// DecodeCertBytesPKCS7DER reads a []byte with PKCS7 DER encoded certificates (e.g. read
// from a file of a HTTP response body), and returns the contents as a []*x509.Certificate
// and an error if encountered. If you don't know in what format the data is encoded,
// use DecodeCertBytes.
func DecodeCertBytesPKCS7DER(certBytes []byte) ([]*x509.Certificate, error) {
	p7, err := pkcs7.Parse(certBytes)
	if err != nil {
		return nil, err
	}

	certs := p7.Certificates
	if len(certs) == 0 {
		err = errors.New("no certificates found")
	}

	return certs, err
}

// DecodeCertBytesPKCS7PEM reads a []byte with PKCS7 PEM encoded certificates (e.g. read
// from a file of a HTTP response body), and returns the contents as a []*x509.Certificate
// and an error if encountered. If you don't know in what format the data is encoded, use
// DecodeCertBytes.
func DecodeCertBytesPKCS7PEM(certBytes []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	pemBytes := certBytes
	for {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if bytes.Equal(rest, pemBytes) {
			return nil, errors.New("not valid PKCS7 PEM data")
		}

		p7, err := pkcs7.Parse(block.Bytes)
		if err != nil {
			return nil, err
		}

		certs = append(certs, p7.Certificates...)
		pemBytes = rest
	}

	var err error
	if len(certs) == 0 {
		err = errors.New("no certificates found")
	}

	return certs, err
}

// DecodeCertBytesPKCS12 reads a []byte with PKCS12 encoded certificates (e.g. read
// from a file of a HTTP response body) and a password. It returns the contents as
// a []*x509.Certificate  and an error if encountered. If you don't know in what
// format the data is encoded, use DecodeCertBytes.
func DecodeCertBytesPKCS12(certBytes []byte, password string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	_, cert, caCerts, err := pkcs12.DecodeChain(certBytes, password)
	if err != nil {
		return nil, err
	} else {
		certs = append(certs, cert)
		certs = append(certs, caCerts...)
	}

	if len(certs) == 0 {
		err = errors.New("no certificates found")
	}

	return certs, err
}

// DecodeCertFile reads a file with DER or PEM encoded certificates and returns
// the contents as a []*x509.Certificate and an error if encountered.
func DecodeCertFile(certFile, password string) ([]*x509.Certificate, error) {
	certBytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	return DecodeCertBytes(certBytes, password)
}

// DecodeKeyBytes reads a []byte with a key and returns a *pem.Block and
// an error if encountered.
func DecodeKeyBytes(keyBytes []byte, password string) (*pem.Block, error) {
	var block *pem.Block
	var err error
	var errStrs []string

	for {
		block, err = DecodeKeyBytesPKCS1(keyBytes)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		block, err = DecodeKeyBytesPKCS8(keyBytes, password)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		block, err = DecodeKeyBytesPKCS12(keyBytes, password)
		if err != nil {
			errStrs = append(errStrs, err.Error())
		} else {
			break
		}

		break
	}

	if err != nil {
		return nil, errors.New(strings.Join(errStrs, "   >>   "))
	}

	return block, nil
}

// DecodeKeyBytesPKCS1 reads a []byte with a PKCS1 PEM encoded key and returns
// a *pem.Block and an error if encountered. If you don't know in what format
// the data is encoded, use DecodeKeyBytes.
func DecodeKeyBytesPKCS1(keyBytes []byte) (*pem.Block, error) {
	if !strings.Contains(string(keyBytes), "-----BEGIN") {
		return nil, errors.New("not a PEM key")
	}
	if strings.Contains(string(keyBytes), "-----BEGIN ENCRYPTED") {
		return nil, errors.New("encrypted key")
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil || !strings.Contains(block.Type, "PRIVATE KEY") {
		return nil, errors.New("failed to decode private key")
	}

	return block, nil
}

// DecodeKeyBytesPKCS8 reads a []byte with an encrypted PKCS8 PEM encoded key and returns
// a *pem.Block and an error if encountered. If you don't know in what format the data
// is encoded, use DecodeKeyBytes.
func DecodeKeyBytesPKCS8(keyBytes []byte, password string) (*pem.Block, error) {
	if !strings.Contains(string(keyBytes), "-----BEGIN") {
		return nil, errors.New("not a PEM key")
	}
	if !strings.Contains(string(keyBytes), "ENCRYPTED") {
		return nil, errors.New("unencrypted key")
	}

	block, _ := pem.Decode(keyBytes)
	parsedKey, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(password))
	if err != nil {
		return nil, err
	}

	return getPKCS8PEMBlock(parsedKey)
}

// DecodeKeyBytesPKCS12 reads a []byte with an encrypted PKCS12 encoded key and returns
// a *pem.Block and an error if encountered. If you don't know in what format the data
// is encoded, use DecodeKeyBytes.
func DecodeKeyBytesPKCS12(keyBytes []byte, password string) (*pem.Block, error) {
	parsedKey, _, _, err := pkcs12.DecodeChain(keyBytes, password)
	if err != nil {
		return nil, err
	}

	return getPKCS8PEMBlock(parsedKey)
}

// DecodeKeyFile reads a file with PEM encoded key and returns the contents as a *pem.Block
// and an error if encountered.
func DecodeKeyFile(keyFile string, password string) (*pem.Block, error) {
	keyBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return DecodeKeyBytes(keyBytes, password)
}

// EncodeCertAsPKCS1PEM converts *x509.Certificate to a []byte with
// data encoded as PKCS1 PEM and an error.
func EncodeCertAsPKCS1PEM(cert *x509.Certificate) ([]byte, error) {
	if cert == nil {
		return nil, errors.New("no certificate found")
	}

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	var buf bytes.Buffer
	err := pem.Encode(&buf, block)

	return buf.Bytes(), err
}

// EncodeKeyAsPKCS1PEM converts *pem.Block private key to a []byte with
// data encoded as PKCS1 PEM and an error.
func EncodeKeyAsPKCS1PEM(key *pem.Block) ([]byte, error) {
	if key == nil {
		return nil, errors.New("no key found")
	}

	var buf bytes.Buffer
	err := pem.Encode(&buf, key)
	return buf.Bytes(), err
}

// FindLeaf looks for the leaf certificate in a chain, this being the
// farthest certificate from the Root CA (usually the certificate of
// a server). It takes a []*x509.Certificate as chain with cert and
// it returns a *x509.Certificate as leaf and an error if zero or
// more than one leaf could be found.
func FindLeaf(certs []*x509.Certificate) (*x509.Certificate, error) {
	candidates := make(map[string]bool)
	var found *x509.Certificate
	for _, cert := range certs {
		if cert.IsCA {
			continue
		}
		found = cert
		candidates[cert.Subject.String()+cert.Subject.SerialNumber] = true
	}

	switch len(candidates) {
	case 0:
		return nil, errors.New("no leaf found")
	case 1:
		return found, nil
	default:
		return nil, errors.New("more than one leaf found")
	}
}

// IsRootCA returns for a given *x509.Certificate true if
// the CA is marked as IsCA and the Subject and the Issuer
// are the same.
func IsRootCA(cert *x509.Certificate) bool {
	return cert.IsCA && cert.Subject.String() == cert.Issuer.String()
}

// SortCerts sorts a []*x509.Certificate from leaf to root CA, or the other
// way around if a the supplied boolean is set to true. Double elements are
// removed.
func SortCerts(certs []*x509.Certificate, reverse bool) []*x509.Certificate {
	chainAsCerts, certByName, order := SortCertsAsChains(certs, reverse)

	var orderedFromLeaves []*x509.Certificate
	var orderedNoLeaves []*x509.Certificate
	for _, subj := range order {
		if ! certByName[subj].IsCA {
			orderedFromLeaves = append(orderedFromLeaves, chainAsCerts[subj]...)
		} else {
			orderedNoLeaves = append(orderedNoLeaves, chainAsCerts[subj]...)
		}
	}

	var ordered []*x509.Certificate
	tmpOrdered := append(orderedFromLeaves, orderedNoLeaves...)
	seen := make(map[string]bool)
	for _, cert := range tmpOrdered {
		if _, ok := seen[cert.Subject.String()]; ok {
			continue
		}
		ordered = append(ordered, cert)
		seen[cert.Subject.String()] = true
	}

	return ordered
}

// SortCertsAsChains sorts a []*x509.Certificate from leaf to root CA, or the other
// way around if a the boolean parameter is set to true. The function returns three
// elements: a map[string][]*x509.Certificate with the subject as key and the chain as
// value, a map[string]*x509.Certificate with the the subject as key and the
// corresponding *x509.Certificate as value and a []string with Subjects that are used
// as keys in the first map, in the order the where found in the given []*x509.Certificate
// parameter.
func SortCertsAsChains(
	certs []*x509.Certificate, reverse bool) (map[string][]*x509.Certificate, map[string]*x509.Certificate, []string) {
	// Get the information needed to follow the chain
	var certByNameOrder []string
	issuerName := make(map[string]string)
	certByName := make(map[string]*x509.Certificate)
	isLeaf := make(map[string]bool)
	for _, cert := range certs {
		subj := cert.Subject.String()
		issuer := cert.Issuer.String()
		if _, ok := certByName[subj]; ok {
			continue
		}
		if !cert.IsCA {
			isLeaf[subj] = true
		}
		certByName[subj] = cert
		issuerName[subj] = issuer
		certByNameOrder = append(certByNameOrder, subj)
	}

	// Create chains
	var order []string
	chain := make(map[string][]string)
	skip := make(map[string]bool)
	for subj, issuer := range issuerName {
		if _, ok := skip[subj]; ok {
			continue
		}

		skip[issuer] = true // we follow the issuers below
		chain[subj] = []string{subj}
		order = append(order, subj)
		presentIssuer := issuer
		for {
			if _, ok := certByName[subj]; !ok {
				continue
			}

			tmpChain := []string{}
			tmpChain = append(tmpChain, chain[subj]...)
			tmpChain = append(tmpChain, presentIssuer)
			skip[presentIssuer] = true
			chain[subj] = tmpChain
			delete(chain, presentIssuer)

			if nextIssuer, ok := issuerName[presentIssuer]; ok {
				if nextIssuer == presentIssuer { // end of this chain
					break
				}

				presentIssuer = nextIssuer
				continue
			}
			break
		}
	}

	chainAsCerts := make(map[string][]*x509.Certificate)
	for subj, chainElems := range chain {
		var ordered []*x509.Certificate
		for _, chainElem := range chainElems {
			if cert, ok := certByName[chainElem]; ok {
				ordered = append(ordered, cert)
			}
		}
		if reverse {
			var reversed []*x509.Certificate
			for idx := len(ordered) - 1; idx >= 0; idx-- {
				reversed = append(reversed, ordered[idx])
			}
			ordered = reversed
		}
		chainAsCerts[subj] = ordered
	}

	return chainAsCerts, certByName, order
}

// SplitCertsAsTree returns a *CertTree where the given certificates
// are assigned as Certificate, Intermediates and Roots. The starting
// leaf certificate must be the first element of the given
// []*x509.Certificate.
func SplitCertsAsTree(certs []*x509.Certificate) *CertTree {
	if len(certs) == 0 {
		return nil
	}

	ordered := SortCerts(certs, false)
	var roots, inters []*x509.Certificate
	for _, cert := range ordered[1:] {
		if IsRootCA(cert) {
			roots = append(roots, cert)
		} else {
			inters = append(inters, cert)
		}
	}

	certTree := CertTree{
		Certificate:   ordered[0],
		Intermediates: inters,
		Roots:         roots,
	}

	return &certTree
}

// VerifyChain verifies the chain of a certificate as part of a CertTree. When the
// Roots field is nil, the OS trust store is used. The function return a boolean with
// the verification result and an string with an associated message with the reason
// of a negative result.
func VerifyChain(tree *CertTree) (bool, string) {
	rootPool := x509.NewCertPool()
	for _, cert := range tree.Roots {
		rootPool.AddCert(cert)
	}

	interPool := x509.NewCertPool()
	for _, cert := range tree.Intermediates {
		interPool.AddCert(cert)
	}

	var verifyOptions x509.VerifyOptions
	if len(rootPool.Subjects()) != 0 {
		verifyOptions.Roots = rootPool
	}
	if len(interPool.Subjects()) != 0 {
		verifyOptions.Intermediates = interPool
	}

	if _, err := tree.Certificate.Verify(verifyOptions); err != nil {
		return false, err.Error()
	}

	return true, ""
}

// VerifyCertAndKey verifies that a certificate (*x509.Certificate) and a key (*pem.Block)
// match, returning the result as a bool.
func VerifyCertAndKey(cert *x509.Certificate, key *pem.Block) bool {
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	keyPEM := pem.EncodeToMemory(key)

	_, err := tls.X509KeyPair(certPEM, keyPEM)
	if err == nil {
		return true
	}
	return false
}

// getPKCS8PEMBlock is used to return a *pem.Block with the correct type
// (so it can be used as reliable metadata).
func getPKCS8PEMBlock(parsedKey interface{}) (*pem.Block, error) {
	var parsedBytes []byte
	var err error
	var blockType string
	switch key := parsedKey.(type) {
	case *rsa.PrivateKey:
		parsedBytes, err = x509.MarshalPKCS8PrivateKey(key)
		blockType = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		parsedBytes, err = x509.MarshalPKCS8PrivateKey(key)
		blockType = "EC PRIVATE KEY"
	case ed25519.PrivateKey:
		parsedBytes, err = x509.MarshalPKCS8PrivateKey(key)
		blockType = "EC PRIVATE KEY"
	default:
		err = errors.New("unknown signature algorithm of private key")
	}
	if err != nil {
		return nil, err
	}

	pemBlock := pem.Block{
		Type:  blockType,
		Bytes: parsedBytes,
	}
	return &pemBlock, nil
}
