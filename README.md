http://localhost:8090/static/
- POSTs data to http://localhost:8090/bin/myapp/123
- Quotas can be found in `./db/app/quotas.json`

# Common
- QuotaManagedWriter
- storeQuotaManager

# Under the hood
- common.go: `maxUploadFileSize = 100 << 20` is the maximum allowed file size, currently 100Mb.
