#!/bin/bash
wget https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-linux-x86_64-1.12.0.tar.gz

echo 'Untarring file'

tar -C /usr/local -xzf libtensorflow-cpu-linux-x86_64-1.12.0.tar.gz

echo 'Running ldconfig'

ldconfig
