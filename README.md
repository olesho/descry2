# Descry v.2 #

Descry is a simple proxy server implementing interception of HTTP/HTTPS requests. It allows building simple XML patterns for data extraction from remote sources.

## Docker setup ##

### Build:

```
cd proxy
docker build -t descry .
```

### Run:

Map "/go/src/app/patterns" dir in container to your patterns directory.
Map port.
```
docker run -v ~/work/src/github.com/olesho/descry2/patterns:/go/src/app/patterns -p 5000:5000 --name descry --rm descry
```

## How to run tests:

## ENV variables:

* PORT="5000" # Listen port. Default: 5000
* PATTERNS_DIR="patterns" # Directory with XML patterns. Default value: 'patterns'
* LOG="error.log" # Log file. STDOUT is used if value empty.

## Usage:

1. Create XML pattern and put into your patterns directory. Pattern examples (for Craiglist and Amazon) you can find in in "patterns" directory.
2. Reload patterns by simply running HTTP GET request to /
3. Use as a proxy: running HTTP/HTTPS request via this proxy will return JSON with data fields. For example this CURL request:

```
curl -x http://localhost:5000 https://sacramento.craigslist.org/search/csr -k
```

will return JSON data containing all positions list.

### Author ###
Oleh Luchkiv
https://github.com/olesho
