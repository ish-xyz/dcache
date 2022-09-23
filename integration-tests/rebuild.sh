pushd ../
docker build -t dpc .
popd
docker-compose down
docker-compose up -d
docker logs -f integration-tests_node1_1
