# Dispatcher

Dispatcher is a tool designed for collecting user requests and saving them to data files which are used for subsequent load testing.

It proxies HTTP requests to the target server and returns the server’s response to the client. When the server responds with a `2xx` status code, Dispatcher saves the proxied request to structured data files for future load testing. You can use flags to override the rules and level of detail for the data to save. The requests are saved to separate files within the Dispatcher local directory:
* **uri.payload** contains `GET` requests.
* **uripost.payload** contains `POST` requests.
* **raw.payload** and **httpjson.payload** contain requests for all methods.

To start using Dispatcher, launch it with the appropriate flags, e.g.:

```
dispatcher_nix -target 'yandex.ru' -port 8080 -nostatic
```

Alternatively, you can run a file with the following code:

```
go run dispatcher.go -target 'yandex.ru' -port 8080 -nostatic
```

Once Dispatcher is running, send all load test requests to `localhost:8080`. 

Also, navigating to `localhost:8080` in your browser will take you to the website specified in the `target` flag.

You can also run the tool without the `target` flag. In this case, the address specified in the request's `Host` header will be used as the target server. 

### Starting flags 

* `port`: Port used to run the server. The default value is `8888`.
* `target`: Address of the service for which requests are proxied and saved.
* `ssl`: Proxies requests through an HTTPS connection.
* `noproxy`: Saves requests without proxying.
* `saveall`: Saves all requests regardless of the proxy response status code.
* `nostatic`: Disables saving requests for static content, such as `css`, `js`, `jpeg`, `jpg`, `png`, `gif`, `ico`, `svg`, `woff`, and `woff2`.
