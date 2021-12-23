
const child_process = require('child_process');
const fs = require('fs');
const http = require('http');
const stream = require("stream");
const site = fs.readFileSync('./index.html');
let golang = undefined;

const server = http.createServer(function (req, res) {
  res.statusCode = 200;
  if (req.url.startsWith("/start-video")) {
    let d = "";
    req.on("data", function (data) {
      d += data.toString();
    });

    req.on("end", function () {
	if (golang !== undefined) {
		golang.kill();
	}
    	golang = child_process.spawn("/home/pi/Projects/webrtc/main", [], {
		cwd: "/home/pi/Projects/webrtc/",
	}, function () {
		console.log("the process finished");
      	});
	golang.on("error", function (err) {
		console.log("there was some error!", err);
	});
	golang.stderr.on("data", function (o) {
		console.log(o.toString());
	});
	golang.stdout.on("error", function () {
		console.log("there was an error");
	});
	golang.stdin.on("error", function () {
		console.log("stdin error");
	});
	golang.stdout.on("data", function (o) {
	    console.log(o.toString());
	    const p = o.toString();
	    const l = p.split("\n");
	    for (let i = 0; i < l.length; i++) {
		    if (l[i].length > 50) {
			res.end(l[i]);
		    }
	    }
	});
	golang.stdin.write(d);
	golang.stdin.end();
    });
    return;
  } else if (req.url === "/stop-video") {
    if (golang !== undefined) {
	    golang.kill();
	    golang = undefined;
    }
    res.end();
    return;
  } else if (req.url === "/shutdown") {
	  console.log("in here?");
	var shutdown = child_process.exec("sudo shutdown 0", function (err) {
		if (err) {
			console.error(err);
		} else {
			console.log("the shutdown ran!");
		}
	});
	  shutdown.stdout.on("data", function (o) {
		console.log(o.toString());
	  })
	return;
  }
  res.setHeader('Content-Type', 'text/html');
  res.end(site);
});

server.listen(1234, '0.0.0.0', () => {
  console.log(`Server running`);
});

