# cloudantsnap

Super-simple Cloudant snapshotting tool for creating incremental snapshots of the winning revisions of documents in an IBM Cloudant database.

- winning revisions only of documents and design documents
- no deletions  (unless `--deletions true` is supplied).
- no attachments
- no conflicts

## Installation

You will need to [download and install the Go compiler](https://go.dev/doc/install). Clone this repo then:

```sh
go build ./cmd/cloudantsnap
```

The copy the resultant binary `cloudantsnap` (or `cloudantsnap.exe` in Windows systems) into your path.

## Configuration

`cloudantsnap` authenticates with your chosen Cloudant service using environment variables as documented [here](https://github.com/IBM/cloudant-go-sdk/blob/v0.10.8/docs/Authentication.md#authentication-with-environment-variables) e.g.

```sh
CLOUDANT_URL=https://xxxyyy.cloudantnosqldb.appdomain.cloud
CLOUDANT_APIKEY="my_api_key"
```

## Usage

Create a snapshot:

```sh
$ cloudantsnap --db mydb
spooling changes for mydb since 0
mydb-snapshot-2022-11-09T160406.195Z.jsonl
mydb-meta.json
```

At a later date, another snapshot can be taken:

```sh
$ cloudantsnap --db mydb
spooling changes for mydb since 23597
mydb-snapshot-2022-11-09T160451.041Z.jsonl
mydb-meta.json
```

Ad infinitum.

You may elect to include deleted documents by adding `--deletions` e.g.

```sh
$ cloudantsnap --db mydb --deletions
...
```

## Finding a document's history

For a known document id e.g. `abc123`:

```sh
grep -h "abc123" mydb-snapshot-*
```

or use a tool to "query" documents matching a selector, such as [mangogrep](https://www.npmjs.com/package/mangogrep):

```sh
cat mydb-snapshot* | mangogrep --selector '{"country": "IN","population":{"$gt":5000000}}'
```

## Restoring a database

Each backup file contains one document per line so we can feed this data to [cloudantimport](https://www.npmjs.com/package/cloudantimport). To ensure that we insert the newest data first, we can concatenate the snapshots in newest-first order into a new, empty database:

```sh
# list the files in reverse time order, "cat" them and send them to cloudantimport
ls -t mydb-snapshot-* | xargs cat | cloudantimport --db mydb2 
# or use "tac" to reverse the order of each file
ls -t mydb-snapshot-* | xargs tac | cloudantimport --db mydb2 
```

Some caveats:

1. This only restores to a new empty database.
2. Deleted documents are neither backed-up nor restored (unless `--deletions` is supplied).
3. The restored documents will have a new `_rev` token. e.g. `1-abc123`. i.e. the restored database would be unsuitable for a replicating relationship with the original database (as they have different revision histories).
4. Attachments are neither backed-up or restored.
5. Conflicting document revisions are neither backed-up nor restored.
6. Secondary index definitions (in design documents) are backed up but will need to be rebuilt on restore.
