#!/bin/bash
set -e

# tar up the files
tar -zcvf cf-orgs-usage.tgz ../../*.go ../../manifest.yml ../../vendor/* ../../README.md ../../LICENSE ../../auditor.sh

