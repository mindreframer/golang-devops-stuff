package main

import (
	"bytes"
	"code.google.com/p/go.crypto/openpgp"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
	"text/template"
)

const jaegerTemplateExtension = ".jgrt"
const jaegerDBExtension = ".jgrdb"
const jaegerDescription = "Jaeger - Template injection program\n\nJaeger is a JSON encoded GPG encrypted key value store. It is useful for separating development with operations and keeping configuration files secure."
const jaegerQuote = "\"Stacker Pentecost: Haven't you heard Mr. Beckett? The world is coming to an end. So where would you rather die? Here? Or in a Jaeger!\" - Pacific Rim"
const jaegerRecommendedUsage = "RECOMMENDED:\n    jaeger -i file.txt.jgrt\n\nThis will run Jaeger with the default options and assume the following:\n    JSON GPG database file: file.txt.jgrdb\n    Output file: file.txt\n    Keyring file: ~/.gnupg/jaeger_secring.gpg\n    No passphrase"

var debug debugging = false

type debugging bool

func (d debugging) Printf(format string, args ...interface{}) {
	// From: https://groups.google.com/forum/#!msg/golang-nuts/gU7oQGoCkmg/BNIl-TqB-4wJ
	if d {
		log.Printf(format, args...)
	}
}

type Data struct {
	Properties []Property
}

type Property struct {
	Name           string //`json:"Name"`
	EncryptedValue string //`json:"EncryptedValue"`
}

func main() {
	// Define flags
	var (
		debugFlag         = flag.Bool("d", false, "Enable Debug")
		inputTemplate     = flag.String("i", "", "Input Template file. eg. file.txt.jgrt")
		jsonGPGDB         = flag.String("j", "", "JSON GPG database file. eg. file.txt.jgrdb")
		outputFile        = flag.String("o", "", "Output file. eg. file.txt")
		keyringFile       = flag.String("k", "", "Keyring file. Secret key in ASCII armored format. eg. secret.asc")
		passphraseKeyring = flag.String("p", "", "Passphrase for keyring. If this is not set the passphrase will be blank or read from the environment variable PASSPHRASE.")
	)

	flag.Usage = func() {
		fmt.Printf("%s\n%s\n\n%s\n\n", jaegerDescription, jaegerQuote, jaegerRecommendedUsage)
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *debugFlag {
		debug = true
	}

	if *inputTemplate == "" {
		flag.Usage()
		log.Fatalf("\n\nError: No input template file specified")
	}

	basefilename := ""

	if strings.HasSuffix(*inputTemplate, jaegerTemplateExtension) {
		basefilename = strings.TrimSuffix(*inputTemplate, jaegerTemplateExtension)
	}

	if *jsonGPGDB == "" {
		if basefilename == "" {
			flag.Usage()
			log.Fatalf("\n\nERROR: No JSON GPG DB file specified or input file does not have a %v extension", jaegerTemplateExtension)
			return
		}
		// Set from the basefilename
		*jsonGPGDB = fmt.Sprintf("%v%v", basefilename, jaegerDBExtension)
	}

	if *outputFile == "" {
		if basefilename == "" {
			flag.Usage()
			log.Fatalf("\n\nERROR: No Output file specified or input file does not have a %v extension", jaegerTemplateExtension)
			return
		}
		// Set from the basefilename
		*outputFile = basefilename
	}

	if *passphraseKeyring == "" {
		passphrase := os.Getenv("PASSPHRASE")
		if len(passphrase) != 0 {
			*passphraseKeyring = passphrase
		}
	}

	debug.Printf("basefilename:", basefilename)
	debug.Printf("jsonGPGDB:", *jsonGPGDB)
	debug.Printf("outputFile:", *outputFile)
	debug.Printf("passphrase:", *passphraseKeyring)
	debug.Printf("keyringFile:", *keyringFile)

	// Read armored private key or default keyring into type EntityList
	// An EntityList contains one or more Entities.
	// This assumes there is only one Entity involved
	// TODO: Support to prompt for passphrase

	var entity *openpgp.Entity
	var entitylist openpgp.EntityList

	if *keyringFile == "" {
		entity, entitylist = processSecretKeyRing()
	} else {
		entity, entitylist = processArmoredKeyRingFile(keyringFile)
	}

	entity = decryptPrivateKeyRing(passphraseKeyring, entity)

	p := make(map[string]string)
	p, err := parseJaegerDBFile(jsonGPGDB, entitylist)
	if err != nil {
		log.Fatal(err)
	}

	if err := writeOutputFile(inputTemplate, outputFile, p); err == nil {
		fmt.Println("Wrote file:", *outputFile)
	}

}

func decodeBase64EncryptedMessage(s string, keyring openpgp.KeyRing) string {
	// Decrypt base64 encoded encrypted message using decrypted private key
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		log.Fatalln("ERR:", err)
	}
	debug.Printf("keyring: #%v", keyring)
	md, err := openpgp.ReadMessage(bytes.NewBuffer(dec), keyring, nil, nil)
	if err != nil {
		log.Fatalln("ERR: Error reading message - ", err)
	}

	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	debug.Printf("md:", string(bytes))
	return string(bytes)
}

func processSecretKeyRing() (entity *openpgp.Entity, entitylist openpgp.EntityList) {
	// Get default secret keyring location
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	jaegerSecretKeyRing := fmt.Sprintf("%v/.gnupg/jaeger_secring.gpg", usr.HomeDir)
	secretKeyRing := ""

	if _, err := os.Stat(jaegerSecretKeyRing); err == nil {
		secretKeyRing = jaegerSecretKeyRing
	} else {
		secretKeyRing = fmt.Sprintf("%v/.gnupg/secring.gpg", usr.HomeDir)
	}

	debug.Printf("secretKeyRing file:", secretKeyRing)
	secretKeyRingBuffer, err := os.Open(secretKeyRing)
	if err != nil {
		panic(err)
	}
	entitylist, err = openpgp.ReadKeyRing(secretKeyRingBuffer)
	if err != nil {
		log.Fatal(err)
	}

	entity = entitylist[0]
	debug.Printf("Private key default keyring:", entity.Identities)

	return entity, entitylist
}

func processArmoredKeyRingFile(keyringFile *string) (entity *openpgp.Entity, entitylist openpgp.EntityList) {
	keyringFileBuffer, err := os.Open(*keyringFile)
	if err != nil {
		log.Fatalln("ERROR: Unable to read keyring file")
	}
	entitylist, err = openpgp.ReadArmoredKeyRing(keyringFileBuffer)
	if err != nil {
		log.Fatal(err)
	}
	entity = entitylist[0]
	debug.Printf("Private key from armored string:", entity.Identities)

	return entity, entitylist
}

func decryptPrivateKeyRing(passphraseKeyring *string, entity *openpgp.Entity) *openpgp.Entity {
	// Decrypt private key using passphrase
	passphrase := []byte(*passphraseKeyring)
	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		debug.Printf("Decrypting private key using passphrase")
		err := entity.PrivateKey.Decrypt(passphrase)
		if err != nil {
			log.Fatalln("ERROR: Failed to decrypt key using passphrase. Make sure you specify a passphrase if required.")
		}
	}
	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil && subkey.PrivateKey.Encrypted {
			err := subkey.PrivateKey.Decrypt(passphrase)
			if err != nil {
				log.Fatalln("ERROR: Failed to decrypt subkey")
			}
		}
	}
	return entity
}

func parseJaegerDBFile(jsonGPGDB *string, entitylist openpgp.EntityList) (map[string]string, error) {
	// json handling
	jsonGPGDBBuffer, err := ioutil.ReadFile(*jsonGPGDB)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Unable to read JSON GPG DB file")
	}

	var j Data
	if err := json.Unmarshal(jsonGPGDBBuffer, &j); err != nil {
		return nil, fmt.Errorf("error:", err)
	}
	debug.Printf("json unmarshal:", j)

	p := make(map[string]string)

	for _, v := range j.Properties {
		debug.Printf("Name: %#v, EncryptedValue: %#v\n", v.Name, v.EncryptedValue)
		p[v.Name] = decodeBase64EncryptedMessage(v.EncryptedValue, entitylist)
	}

	debug.Printf("properties map:", p)
	return p, nil
}

func writeOutputFile(inputTemplate *string, outputFile *string, p map[string]string) error {
	// Template parsing
	t := template.Must(template.ParseFiles(*inputTemplate))

	buf := new(bytes.Buffer)
	t.Execute(buf, p) //merge template ‘t’ with content of ‘p’

	bytes, _ := ioutil.ReadAll(buf)
	debug.Printf(string(bytes))

	// Writing file
	// To handle large files, use a file buffer: http://stackoverflow.com/a/9739903/603745
	if err := ioutil.WriteFile(*outputFile, bytes, 0644); err != nil {
		return fmt.Errorf("error:", err)
	}

	return nil
}
