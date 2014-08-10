## Jaeger

Jaeger is a JSON encoded GPG encrypted key value store. It is useful for generating and keeping configuration files secure. Jaeger is written in Go.

PaaS providers assume configuration settings should be stored as environment variables.  There is nothing inherently wrong with this but it does assume your code base and application is set up to support reading settings from environment variables.

Jaeger simplifies this by providing a system for storing, managing and generating configuration files that applications commonly use.

A basic set of files consists of the following.

* Template file.  A file in [Go text template format](http://golang.org/pkg/text/template/) which holds the structure of the generated file.
 * Jaeger assumes the template file name is in the form `filename.txt.jgrt` where `filename.txt` is the file name of the generated file.
* JSON encoded GPG encrypted key value store file.  This file is managed using the `jaegerdb` program.
 * Jaeger assumes the template file name is in the form `filename.txt.jgrdb` where `filename.txt` is the file name of the generated file.

The best way to experience Jaeger is to run through the Quickstart below.

Jaeger uses the Go standard library except for the `code.google.com/p/go.crypto/openpgp` package.

> Stacker Pentecost: Haven't you heard Mr. Beckett? The world is coming to an end. So where would you rather die? Here? Or in a Jaeger!
>
> -- <cite>Pacific Rim</cite>


## Quickstart

### Installation

Install Jaeger:

    go get github.com/jyap808/jaeger/jaeger
    go get github.com/jyap808/jaeger/jaegerdb

This will create the binaries `jaeger` and `jaegerdb` in your `$GOPATH/bin` directory.

### Make standalone public and private GPG keys

Jaeger can also read public and private keys in ASCII armored format via command line flags.  Here we are creating standalone GPG keys that Jaeger will use by default.

    gpg --keyring ~/.gnupg/jaeger_pubring.gpg --secret-keyring ~/.gnupg/jaeger_secring.gpg --gen-key --no-default-keyring

Select:

    (1) RSA and RSA (default)

Select:

    What keysize do you want? 2048

Select: 

    Key is valid for? 0 

Enter:

    Real name: Jaeger Test
    Email address: jaeger@example.com
    Comment:                         

Select:

    (O)kay

Set a passphrase for the private key. eg. 'test passphrase'


### Verify your keys were created OK

Secret key:

    gpg --no-default-keyring --list-secret-keys --secret-keyring ~/.gnupg/jaeger_secring.gpg 

Public key:

    gpg --no-default-keyring --list-keys --keyring ~/.gnupg/jaeger_pubring.gpg

### Create a test Template file

Create `test.txt.jgrt` with contents:

    datebase.username = dbuser
    database.password = {{.DatabasePassword}}
    field2.user = user2
    field2.password = {{.Field2}}

### Create an empty JSON GPG database

Jaeger can also read public and private keys in ASCII armored format via command line flags.

    jaegerdb -init -j test.txt.jgrdb

### Add some properties and values

    jaegerdb -j test.txt.jgrdb -a DatabasePassword -v "This is the database password"

    jaegerdb -j test.txt.jgrdb -a Field2 -v "This is field 2"

Take a look at the database file.  Note that the values are encrypted:

    cat test.txt.jgrdb

### Generate a file

    jaeger -i test.txt.jgrt -p "test passphrase"

Take a look at the generated file.  Note that the decrypted values have now been injected:

    cat test.txt

### Change a property value

    jaegerdb -j test.txt.jgrdb -c DatabasePassword -v "This is the NEW database password"

### Regenerate the file

    jaeger -i test.txt.jgrt -p "test passphrase"

Take a look at the newly generated file.  Note that the generated file now has the new property value:

    cat test.txt

## More options

Use `jaeger -h` and `jaegerdb -h` to list all options.


## License

Copyright (c) 2014 Julian Yap

[MIT License](https://github.com/jyap808/jaeger/blob/master/LICENSE)
