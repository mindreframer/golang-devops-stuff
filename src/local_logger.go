package main

import (
	"bufio"
	"os"
	"time"

	"github.com/Sendhub/logserver/logger"
)

func (this *Local) Logger(host, applicationName, process string) error {
	errors := make(chan error)
	c := make(chan []byte)
	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			line, err := r.ReadBytes('\n')
			if err != nil {
				errors <- err
				break
			}
			c <- line
		}
	}()
	throttled := make(chan []byte, 100)
	go func() {
		for bs := range c {
			select {
			case throttled <- bs:
				continue
			default:
			}

			select {
			case <-throttled:
			default:
			}

			select {
			case throttled <- bs:
			default:
			}
		}
		close(throttled)
	}()

	var client *logger.Client
	for {
		select {
		case line := <-throttled:
			var err error
			for {
				if client == nil {
					client, err = logger.Dial(host, applicationName, process)
					if err != nil {
						client = nil
						time.Sleep(time.Second * 5)
						continue
					}
				}
				err = client.Send(line)
				if err != nil {
					client.Close()
					client = nil
					time.Sleep(time.Second * 5)
					continue
				}

				break
			}
		case err := <-errors:
			return err
		}
	}

	return nil
}
