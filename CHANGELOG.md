## Changelog

### v0.8.4

* [read,delete] enable {def,ns,tsds}-{,csv,json} to get multiple keys, this also define multi-val type for "read" output

#### v0.8.3

* ParentNS feature provided, provide Parent NameSpace for all keys (in any key-type) to be used while Push/Read/Delete
* dbtasks moved to use Packet{}, abstract-ing out how data is received on multiple communication channels

#### v0.8.2

* [push] encoding for ns/tsds may contain a root parent-ns config to pre-pend for all ASCII multi-val

---

#### v0.8.1

* added json support
* fixed multi-val support for non-tsds key-types

#### v0.8.0

* huge refactor and design fixes

#### *Good To Have*

>
> * fetch all one-step {,all with val::} children in a particular key namespace, don't fetch all with-val:: children in a particular key namespace, 'cuz that can be done by doing ns and looking at all keys
>
> * (logging) "debug:default" log triggers only at failure status; "verbose" with only DBRest call logged again
>
> * [read] features for tsds like: {latest,this}_{year,month,week,day,hour,hour,min,dot}
>
> * [read] for tsds: mean, median, max, min, from_X_to_Y, radius_X_of_Y, more_than_X, less_than_X, matching_to_X, not_matching_to_X
>

#### *Design Decisions*

>
> * if task-type tokens need to grow to more than 3, move to msgpack for entire packet sent to dbtasks
>
