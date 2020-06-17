#!/bin/bash

# $1 is the name of the original config file for swagger-combine
# this only works with .json files because the input for swagger-combine
# needs to be known
# after this script has run, the documentation can be found on
# localhost:8080/docs/
./swagger_combine_config_preprocess $1
swagger-combine combined-config.json -o combinedOAS.json
./swagger_host/swagger_host -sd swagger_host/swagger-ui/ combinedOAS.json
