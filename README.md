# lilpinger


lilpinger is a small site pinging tool I wrote to notify me via SMS and email if any of the properties I manage are running slow or having connection problems. Pinging each URL happens in a separate goroutine allowing for lags and errors for each URL to be independent of each other.

## Features
- Pings a list of sites once per ping interval in separate go routines
- SMS and email notifications for:
	- connection errors
	- response times over lag threshold
	
## Configuration
All configuration is handled inside lilpinger.toml. The following params can be set:

- **PingInterval**: How often to ping each url, in seconds.
- **LagThreshold**: If response time is slower than this, send alert (in seconds).   
- **URLsFile**: Path to a text file of URLs to ping. Each URL needs to be on a new line. The file path can be a relative or absolute reference.  
- **Twilio**: Credentials for SMS notifications via Twilio.
- **SMTP**: Credentials for email account to send notifications from.
- **Notify**: Mobile phones and emails to notify on ping errors or slow responses.

## Runing lilpinger

### From Go source

```
go run lilpinger
```

You will see output for each url in your URLsFile. 

### Compiled version as a foreground process
This will ouput ping data to the console

```
./lilpinger
```

### Compiled version as a background process
This will create a lilpinger.log file with lilpinger's output

```
./lilpinger > lilpinger.log &
```

## Questions?
Ping me on twitter [@alexrolek](http://twitter.com/alexrolek)

## License

The MIT License (MIT)

Copyright (c) 2013 Tiny Factory, LLC

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
