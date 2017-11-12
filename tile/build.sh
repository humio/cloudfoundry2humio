#!/bin/bash

function archive() {
    rm -rf resources/humio-nozzle.zip 
    zip -r resources/humio-nozzle.zip ../caching ../humio ../nozzle ../vendor ../main.go ../manifest.yml ../CHANGELOG.md ../README.md ../mocks
}

function build() {
    tile build
}

archive
build