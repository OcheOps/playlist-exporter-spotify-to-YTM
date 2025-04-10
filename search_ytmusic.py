import sys
import json
from ytmusicapi import YTMusic

ytmusic = YTMusic("headers_auth.json")  # <- you'll still need this file once

query = sys.argv[1]
results = ytmusic.search(query, filter="songs")

if results:
    print(json.dumps(results[0]))  # return best match
else:
    print("{}")  # not found