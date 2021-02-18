# store

KOTS has configurable backing stores. 
These are written at compile time, and must be included in this directory.

To add a new store, implement the entire interface defined in `store_interface.go`.


| Store | Description | 
|-------|-------------|
| [s3pgstore](.) | (Default in KOTS 1.18). Uses S3 object store and postgres for persistence |
