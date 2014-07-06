#!/bin/sh
docker run -p 7000:7000 -v /yourdatadirectory:/data -e READTOKEN=123 -e WRITETOKEN=abc blang/pushr