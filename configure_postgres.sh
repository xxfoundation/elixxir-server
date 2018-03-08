#!/usr/bin/env bash

dropdb --if-exists cmix_server
dropuser --if-exists cmix
createuser cmix
createdb --owner=cmix cmix_server
