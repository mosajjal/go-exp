# pebble-cli
Simple CLI to work with [pebble](https://github.com/cockroachdb/pebble) key/value database 


# Usage

```
Usage:
  app [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  dump        dump all the keys and values in the db
  help        Help about any command
  index       Index the csv from stdin to the database
  query       query a list of keys coming from stdin against db
  remove      remove list of keys coming from stdin

Flags:
  -h, --help          help for app
  -p, --path string   database folder (default "./mydb")

Use "app [command] --help" for more information about a command.
```
