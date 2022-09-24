pushd ../
docker build -t dcache .
popd
docker-compose down
docker-compose up -d
docker logs -f integration-tests_node1_1
