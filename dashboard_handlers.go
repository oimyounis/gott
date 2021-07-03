package gott

import (
	"net/http"

	"github.com/labstack/echo"
)

func initDashboardRoutes(e *echo.Echo, b *Broker) {
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "GOTT Dashboard")
	})

	e.GET("api/topics", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, b.TopicFilterStorage.JSON())
	})

	e.GET("api/clients/:id", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, b.getClientJSON(c.Param("id")))
	})

	e.GET("api/clients", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, b.getClientsJSON())
	})

	e.GET("api/stats", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, b.Stats.JSON())
	})

	e.POST("api/stats/reset", func(c echo.Context) error {
		b.Stats.Reset()
		return c.String(http.StatusOK, "")
	})

	e.GET("/stats", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `
<body>
<button id="resetBtn">Reset</button>
<h4>Server Stats:</h4>
<pre id="stats"></pre>

<br>

<h4>Clients:</h4>
<pre id="clients"></pre>

<br>

<h4>Subscriptions:</h4>
<pre id="topics"></pre>

<script>
var stats = document.querySelector("#stats");
var topics = document.querySelector("#topics");
var clients = document.querySelector("#clients");

document.querySelector("#resetBtn").onclick = function(){
fetch("/api/stats/reset", {method:"POST"});
};

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

fetch("/api/clients").then((res)=>res.json()).then((json)=>{clients.innerHTML=json.join('\n')});
}

setInterval(fetchStats, 1000);
</script>
</body>
`)
	})
}
