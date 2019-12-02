echo "building..."
go build -o ./bin/srv
echo "copying resources..."
cp -r ./resources ./bin/resources
cd ./bin
echo "zipping..."
rm ./srv.zip
zip -r ./srv.zip ./srv ./resources
cd ../
echo "uploading...\n"
echo `curl --upload-file ./bin/srv.zip https://transfer.sh/srv.zip`
echo "\ndone\n"