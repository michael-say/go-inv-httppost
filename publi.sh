echo 'building...\n'
go build -o ./bin/srv
zip -r ./bin/srv.zip ./bin/srv ./resources
echo 'uploading...\n'
echo `curl --upload-file ./bin/srv.zip https://transfer.sh/srv.zip`
echo '\ndone\n'