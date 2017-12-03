# AcoustID Private Catalog API

This AcoustID API allow you to maintain and search in your own catalog of songs.

## Table of Contents

* [Endpoints](#endpoints)
  * [Create Catalog](#create-catalog)
  * [Delete Catalog](#delete-catalog)
  * [Get Catalog Details](#get-catalog-details)
  * [List Catalogs](#list-catalogs)
  * [Add or Update Track](#add-or-update-track)
  * [Delete Track](#delete-track)
  * [Get Track Details](#get-track-details)
  * [Search](#search)
* [Conventions](#conventions)
  * [Authentication](#authentication)
  * [Error Handling](#error-handling)
* [Code Example](#code-example)


## Endpoints


### Create Catalog

Create a new catalog. You usually don't need to call this, since the catalog will be
automatically created when adding the first track. In the future, you will be able to
update catalog settings using this method.

#### Endpoint

    PUT /v1/priv/{catalog}

#### Parameters

None

#### Sample request

    PUT https://api.acoustid.biz/v1/priv/prod-music

#### Sample response

```json
{
  "catalog": "prod-music"
}
```



### Delete Catalog

Delete a catalog and all of its tracks.

#### Endpoint

    DELETE /v1/priv/{catalog}

#### Parameters

None

#### Sample request

    DELETE https://api.acoustid.biz/v1/priv/prod-music

#### Sample response

```json
{
  "catalog": "prod-music"
}
```



### Get Catalog Details

Get details about a catalog.

#### Endpoint

    GET /v1/priv/{catalog}

#### Parameters

None

#### Sample request

    GET https://api.acoustid.biz/v1/priv/prod-music

#### Sample response

```json
{
  "catalog": "prod-music",
  "tracks": {
    "count": 1000
  }
}
```



### List Catalogs

List all your catalogs.

#### Endpoint

    GET /v1/priv

#### Parameters

None

#### Sample request

    GET https://api.acoustid.biz/v1/priv

#### Sample response
 
```json
{
  "catalogs": [
    {"catalog": "prod-music"},
    {"catalog": "test-music"}
  ]
}
```

### Add or Update Track

Add a new track to the catalog, or update an existing track.
You can provide your own track ID or let the system generate one for you.
Every track must have an audio fingerprint. You can optionally also send track metadata, which will be
returned from the search API. You can use it for identification if you don't have your own track IDs.

#### Endpoint

    PUT /v1/priv/{catalog}/{track}
    POST /v1/priv/{catalog}

#### Parameters

| Name | Data Type | Description |
| --- | --- | --- |
| fingerprint | string | Audio fingerprint of the whole song. |
| metadata | complex | JSON object with your own metadata. |
| allow_duplicate | bool | Allow duplicate fingerprint to be added to the catalog. Default: false |

#### Sample request

    PUT https://api.acoustid.biz/v1/priv/prod-music/track-1234

```json
{
  "fingerprint": "AQAAeUmUJEuSTNEIFfnhA9fh...",
  "metadata": {
    "title": "Song title",
    "author": "Song author"
  }
}
```

#### Sample response

```json
{
  "id": "track-1234",
  "catalog": "prod-music"
}
```

### Delete Track

Delete a track from the catalog.

#### Endpoint

    DELETE /v1/priv/{catalog}/{track}

#### Parameters

None

#### Sample request

    DELETE https://api.acoustid.biz/v1/priv/prod-music/track-1234

#### Sample response

```json
{
  "id": "track-1234",
  "catalog": "prod-music"
}
```

### Get Track Details

Get details about a track from the catalog.

#### Endpoint

    GET /v1/priv/{catalog}/{track}

#### Parameters

None

#### Sample request

    GET https://api.acoustid.biz/v1/priv/prod-music/track-1234

#### Sample response

```json
{
  "id": "track-1234",
  "catalog": "prod-music",
  "metadata": {
    "title": "Song title",
    "author": "Song author"
  }
}
```

### Search

Find tracks in the catalog that match the provided audio fingerprint.
This endpoint allows two identification modes.

In stream mode, you send a fingerprint generated
from 10-30 seconds of audio and it will find all tracks that contain the audio. This allow you to
identify tracks in real-time streams, but the short fingerprints can cause false positive matches, so
you should only consider it a match if multiple consecutive searches find the same track.

Alternatively, you can do full track search, where you generate the fingerprint from the entire
audio file. This is both faster and more precise, but it will not find tracks that contain the fingerprint
somewhere in the middle. This is useful for catalog deduplication.

#### Endpoint

    POST /v1/priv/{catalog}/_search

#### Parameters

| Name | Data Type | Description |
| --- | --- | --- |
| fingerprint | string | Audio fingerprint to search for. |
| stream | boolean | Whether this identification of a part of an audio stream, or an song. Default: false |

#### Sample request

    POST https://api.acoustid.biz/v1/priv/prod-music/_search

```json
{
  "stream": true,
  "fingerprint": "AQAAeUmUJEuSTNEIFfnhA9fh..."
}
```

#### Sample response

```json
{
  "catalog": "prod-music",
  "results": [
    {
      "id": "track-1234",
      "metadata": {
        "title": "Song title",
        "author": "Song author"
      },
      "match": {
        "position": 0,
        "duration": 17.580979
      }
    }
  ]  
}
```


## Conventions

### Authentication

All requests need to be authenticated. The API uses [HTTP Basic Authentication](https://en.wikipedia.org/wiki/Basic_access_authentication) with
the username "x-acoustid-api-key" and the password set to your application's API key.
This makes it easy to use using standard HTTP client libraries.

### Error Handling

You can expect the API to return any valid [HTTP status code](https://en.wikipedia.org/wiki/List_of_HTTP_status_codes).
You will only get the documented response when the API returns status code 200.
For other status codes you will get a JSON document in the following structure:

```json
{
  "status": 400,
  "error": {
    "type": "invalid_request",
    "reason": "Invalid track ID"
  }
}
``` 

## Code Example

Using [Python](https://www.python.org/), [requests](http://docs.python-requests.org/en/master/) and
[fpcalc](https://acoustid.org/chromaprint):

```python
import subprocess, requests

audio_file = "path/to/song.wav"
process = subprocess.run(["fpcalc", "-json", "-length", "0", audio_file], stdout=subprocess.PIPE)
output = json.loads(process.stdout.decode('utf8'))

session = requests.Session()
session.auth = ('x-acoustid-api-key', your_api_key)

url = "https://api.acoustid.biz/v1/priv/prod-music/track-1234"
payload = {
  "fingerprint": output["fingerprint"],
  "metadata": {
    "title": "Song title",
    "artist": "Song artist",
  },
} 
rv = session.put(url, json=payload)
rv.raise_for_status()
response = rv.json()
```
