#!/bin/bash
protoc -I controlpanel/ controlpanel/control.proto --go_out=plugins=grpc:controlpanel
