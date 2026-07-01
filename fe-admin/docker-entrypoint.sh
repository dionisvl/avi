#!/bin/sh
set -e

if [ -z "$API_URL" ]; then
    echo "Error: API_URL environment variable is not set" >&2
    exit 1
fi

# Generate env.js from template by substituting only allowed variables
envsubst '${API_URL}' < /etc/nginx/templates/env.js.template > /usr/share/nginx/html/env.js

echo "Generated env.js with API_URL=${API_URL}"

exec nginx -g "daemon off;"
