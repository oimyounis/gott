package dashboard

import (
	"gott"
	"net/http"

	"github.com/labstack/echo"
)

func handle(b *gott.Broker) {
	HTTP.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "GOTT Dashboard")
	})

	HTTP.GET("api/topics", func(c echo.Context) error {
		return c.String(http.StatusOK, b.TopicFilterStorage.String())
	})

	HTTP.GET("api/stats", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, b.Stats.Json())
	})

	HTTP.GET("/stats", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `
<body>
<h4>Server Stats:</h4>
<pre id="stats"></pre>

<br>

<h4>Subscriptions:</h4>
<pre id="topics"></pre>

<script>
var stats = document.querySelector("#stats");
var topics = document.querySelector("#topics");

function formatTime(seconds) {
var time = new Date(1000 * seconds).toISOString().substr(11, 8);
var parts = time.split(":");
return parts[0] + "h " + parts[1] + "m " + parts[2] + "s";
}

function fetchStats() {
fetch("/api/stats").then((res)=>res.json())
.then((data)=>{
	stats.innerHTML= 'Received Messages: '+data.received+
'\nSent Messages: '+data.sent+
'\nSubscriptions: '+data.subscriptions+
'\nBytes In: '+data.bytesIn+
'\nBytes Out: '+data.bytesOut+
'\nConnected Clients: '+data.clients+
'\nUptime: '+formatTime(data.uptime);
});

fetch("/api/topics").then((res)=>res.text()).then((text)=>{topics.innerHTML=text});
}

setInterval(fetchStats, 1000);
</script>
</body>
`)
	})
}
