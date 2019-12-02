go build -o ./bin/srv
zip -r ./bin/srv.zip ./bin/srv ./resources
curl --upload-file ./bin/srv https://transfer.sh/srv