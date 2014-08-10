## GoShare's Concept for Users

A better look into how to use GoShare for your usability scenario.

Learn how to use it over:
* [HTTP]() at [wiki: GoShare over HTTP]()
* [ZeroMQ]() at [wiki: GoShare over ZeroMQ]()

---

## GoShare Request

it's constructed of following pieces

```ASCII

[DB-Action] [Task-Type] ([Time-Dot]) ([Parent-NameSpace]) {[Key ([Val])] OR [DB-Data]}

```

---

### DB-Action

* 'push':   to push provided key-val depending on task-type and Parent-Namespace
* 'read':   to read and return values for provided keys depending on task-type and Parent-Namespace
* 'delete': to delete values for provided keys depending on task-type and Parent-Namespace

---

### Task-Type

Task-Type is a string value telling goshare how to understand the data for the request and how to prepare response as well.

Task-Type string can have maximum 3 tokens, separated by a hyphen '-'. The structure of its is

```ASCII

KeyType-ValType-Helpers

```


#### What all '**KeyType**' key-val can be created?

this helps GoShare understand what kin'of Keys are we dealing with from the following

* 'default': create a 'key' for particula 'value'
* 'ns':      create a 'namespace' with a particular value, reachable via all parents
* 'tsds':    create a 'namespace for provided timedot[1]', reachable like 'ns' filtered from larger time unit to smaller
* 'now':     create a 'namespace for a timedot[1] mapped to time of its getting pushed', reachable like 'tsds'

Here, 'now' KeyType is only usable for 'push' DB-Action.


#### What all '**ValType**' data can be handled from Request?

* same for ZeroMQ and HTTP
>
> * nil:       opts for 'default' format
> * 'default': handles Request-Default-Format[2], used for single key handling
> * 'csv':     handles just one data-string in Request-CSV-format[3], better for multiple keys
> * 'json':    handles just one data-string in Request-JSON-format[4], better for multiple keys
>


#### What all '**Helpers**' are available

it let's you have some sanity around sent DB-Data and prepare it using the value from these helpers

* 'parentNS': this let's you use only child-key-names in DB-Data and send Parent-Namespace for them separately
> for example DB-Data 'name:first,anon' with ParentNS 'people', translates to 'people:name:first,anon' in DB

when used, Second token for Task-Type can't be left blank for default, so use 'default' as ValType


##### Examples:

* 'default':
> push request will just provide a key and val as separate tokens, request depict status
> read/delete request will have just a key, response will be csv of 'key,val'

* 'default:default':
> same like one before it for 'default'

* 'ns-default';
> push request like 'default', but creates all parent namespace required for key
> read/delete like 'default', but will clean up all key-vals under provided key

* 'tsds-json':
> push request will just provide a key-val as dictionary in JSON, request depict status, will also have all fields requird for 'timedot'[1] and keys will be namespace by them
> read/delete request will have keys as list in JSON, response will be JSON of '{"key":"val",..}'

* 'tsds-default-parentNS':
> push request like 'ns-default', will have all fields required for 'timedot'[1] and key will be namespace by them
> read/delete like 'ns-default', just uses key with power of parentNS

* 'ns-csv-parentNS'
> push request like 'ns-csv', will have all fields required for 'timedot'[1] and key will be namespace by them
> read/delete like 'ns-csv', just uses key with power of parentNS

---

## GoShare Response


#### What all '**ValType**' data can be handled for Response?

* for "Push" (Create/Update) and Delete calls
>
> * on ZeroMQ: empty-string for Success, error-string for Failure
> * on HTTP: "Success" for Success, error-string for Failure
>

* for Read calls (same on ZeroMQ and HTTP)
>
> * nil, 'default': opts for 'csv' Response-CSV-Format format
> * 'csv':          handles just one data-string in CSV-format, better for cli apps
> * 'json':         handles just one data-string in JSON-format, better for not-cli apps
>
> ** it uses the same ValType token for response in which the request has been sent **
> ** so if you want response in json, request in json... it's like want respect, better give respect ;) **
>

---

##### Clarifications if required:

>
> * [1] 'timedot':
> > it's an identifiable point in time by provided Year, Month, Day, Hour, Minute and Second.
>
> * [2] 'Request-Default-Format':
> > for '*PUSH*': it's two different fields for Key and Val, as in above ASCII-gram can be seen ```[Key ([Val])]```
> > for '*Read/Delete*': it's just one field for Key ```'[key]'```
>
> * [3] 'Request-CSV-format':
> > for '*PUSH*': one ('key,val') or multiple ('key,val\nkey2,val2') available in CSV format
> > for '*Read/Delete*': one ('key') or multiple ('key1,key2,key3,key4') available in CSV format
>
> * [4] 'Request-JSON-format':
> > for '*PUSH*': one ('{"key":"val"}') or multiple ('{"key":"val","key2":"val2"}') available in JSON format
> > for '*Read/Delete*': one ('["key"]') or multiple ('["key1","key2","key3","key4"]') available in JSON format
>

* [A] Create/Update is same thing for GoShare. Just overwrites.
