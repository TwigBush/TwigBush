package client

import "github.com/TwigBush/gnap-go/internal/gnap"

type KeyPair struct {
	PrivateKey gnap.JWK
	PublicKey gnap.JWK
}

type Proof string

const (
	HTTPSig 	Proof  = "httpsig"
	MTLs    	Proof  = "mtls"
	DPoP		Proof  = "dpop"
    JSECP256k1 	Proof  = "jsecp256k1"
)

type ProofMethod struct {
	HTTPSig Proof 
	MTLs    Proof 
	DPoP    Proof
}

type Configuration struct {
	ClientID 		string
	ClientName 		string
	ClientVersion 	string
	ClientURI 		string
	KeyPair 		KeyPair
	ProofMethod 	ProofMethod 
	AsURL			string
}
