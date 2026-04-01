#!/bin/bash

echo "Cleaning dist ..."
rm -rf ./dist/*

echo "Building project ..."
bun build ./src/app.ts --outdir ./dist --splitting --minify

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "Copying public assets..."
    cp ./logo.svg ./favicon.svg ./dist/
else
    echo "Build failed!"
    exit 1
fi

# ls -lh ./dist
