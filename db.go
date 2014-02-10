/*Mongodb utility methods*/
package hamster

import (
	"log"

	"labix.org/v2/mgo"
)

//mongodb
type Db struct {
	Url        string
	MgoSession *mgo.Session
}

//get new db session
func (d *Db) GetSession() *mgo.Session {
	if d.MgoSession == nil {
		var err error
		d.MgoSession, err = mgo.Dial(d.Url)
		if err != nil {
			log.Fatalf("dialing mongo url %v failed with %v", d.Url, err)
			//panic(err)
		}
	}
	return d.MgoSession.Clone()

}
