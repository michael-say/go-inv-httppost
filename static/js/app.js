const APP = 'mytestapp'
const WORKSPACE = '1003452'

function onError(status, err) {
    $("#log").append($("<li class='error'>"+status+": "+err+"</li>"))
}

function onSuccess(data) {
    var res = JSON.parse(data);
    var html = "<li class='success'>Successfully added file(s). Quota left: <strong>" + res.diskQuota + "</strong>"
    html += "<ol>";
    for (const f of res.result) {
        html += ("<li><a target='_blank' href='/bin/"+APP+"/"+ WORKSPACE + "/"+f.guid+"'>"+f.fileName + "</a></li>")
    }
    html += "</ol></li>";

    $("#log").append($(html))
}

$(document).ready(function(ev) {
    $("#upload-form").on('submit', (function(ev) {
        ev.preventDefault();
        $.ajax({
            xhr: function() {
                var progress = $('.progress'),
                    xhr = $.ajaxSettings.xhr();

                progress.show();

                xhr.upload.onprogress = function(ev) {
                    if (ev.lengthComputable) {
                        var percentComplete = parseInt((ev.loaded / ev.total) * 100);
                        progress.val(percentComplete);
                        if (percentComplete === 100) {
                            progress.hide().val(0);
                        }
                    }
                };

                return xhr;
            },
            url: '/bin/mytestapp/1003452',
            type: 'POST',
            data: new FormData(this),
            contentType: false,
            cache: false,
            processData: false,
            success: function(data, status, xhr) {
                onSuccess(data)
            },
            error: function(xhr, status, error) {
                onError(status, error)
            }
       });
    }));
});