## Using GoShare over HTTP

#### By HTTP Method, send HTTP Request to "<IP>/db" as

* POST/PUT method for "push" db-action
* GET method for "read" db-action
* DELETE method for "delete" db-action

---

#### By Route, send GET method HTTP Requests to

* "<IP>/put" for "push" db-action
* "<IP>/get" for "read" db-action
* "<IP>/del" for "delete" db-action

---

URL form fields can be added for

* "type":     task-type {default,ns.tsds,now}-{,default,csv,json}-{,parentNS}; like ns-json or tsds-csv-parentNS
* "key":      for key to be used in "default" key-type
* "val":      for value to be used in "default" key-type on "push" db-action
* "dbdata" :  to specify data-string for val-types like csv, json
* "parentNS": to specify ParentNamespace for all/any keys to specified in request
* "year", "month", "day", "hour", "min", "sec" to specify all values of a timedot for key-type "tsds"

>
> for POST/PUT HTTP Method on <IP>/db, the field values can be provided via POST Body or URL fields
>

---

#### Examples

all '%s' seen in URLs demand a suitable value for that field there

* read value for a key 'anything' ``` http://<IP>:<PORT>/get?type=default&key=anything ```

* push with full namespace of 'any:thing" value 'huh' ``` http://<IP>:<PORT>/put?type=ns&key=any:thing&val=huh ```

* delete all self and child namespace for 'any' and 'all' ``` http://<IP>:<PORT>/del?type=ns-csv&dbdata=any,all ```

* push 'up' for 'state' as timeseries for given time point ``` http://<IP>:<PORT>/put?key=state&val=up&type=tsds&year=%s&month=%s&day=%s&hour=%s&min=%s&sec=%s ```

* push namespaced-key "name:first" with value "bob" ``` http://<IP>:<PORT>/put?dbdata={\"name:first\":\"bob\"}&type=ns-json ```

* push timeseries value for timedot of being stored of key 'a' value 'A', key 'b' value 'B'  ``` http://<IP>:<PORT>/put?dbdata=a,A%0D%0Ab,B&type=now-csv ```

* to use a parent-namespace helper (prepend all keys with it) for keys ``` <ANY-OTHER-URL>&parentNS=%s ```

---
