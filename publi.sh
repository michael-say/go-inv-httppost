echo "building..."
go build -o ./bin/srv
echo "copying resources..."
cp ./resources ./bin/resources
cd ./bin
echo "zipping..."
rm ./srv.zip
zip -r ./srv.zip *
echo "uploading...\n"
echo `curl --upload-file ./bin/srv.zip https://transfer.sh/srv.zip`
cd ../
echo "\ndone\n"