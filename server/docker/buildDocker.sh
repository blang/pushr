#!/bin/sh
go build ../
docker build -t blang/pushr .