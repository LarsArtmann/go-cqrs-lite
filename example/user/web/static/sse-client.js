class CQRSEventSource {
  constructor(url, options = {}) {
    this.url = url;
    this.reconnectInterval = options.reconnectInterval || 3000;
    this.eventHandlers = new Map();
    this.es = null;
    this.connected = false;
  }

  connect() {
    const clientId = crypto.randomUUID();
    const url = `${this.url}?client=${clientId}`;

    this.es = new EventSource(url);

    this.es.onopen = () => {
      this.connected = true;
      console.log("[cqrs-sse] connected:", this.url);
    };

    this.es.onerror = () => {
      this.connected = false;
      console.warn("[cqrs-sse] connection lost, reconnecting...");
    };

    this.es.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const eventType = event.type || data.type;

      if (this.eventHandlers.has(eventType)) {
        this.eventHandlers.get(eventType)(data);
      }

      if (this.eventHandlers.has("*")) {
        this.eventHandlers.get("*")(data, eventType);
      }
    };

    return this;
  }

  on(eventType, handler) {
    this.eventHandlers.set(eventType, handler);
    return this;
  }

  close() {
    if (this.es) {
      this.es.close();
      this.connected = false;
    }
  }
}

// Usage:
//   const client = new CQRSEventSource("/events").connect();
//   client.on("UserCreated", (data) => console.log("User created:", data));
//   client.on("*", (data, type) => console.log(`Event [${type}]:`, data));
