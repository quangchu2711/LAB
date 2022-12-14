#include <WiFi.h>
#include <PubSubClient.h>
#include "DHT.h"
#include "ArduinoJson.h"

const char* ssid = "Xuong_1";
const char* password = "68686868";

#define MQTT_SERVER "192.168.0.102"
#define MQTT_PORT 1883
#define MQTT_USER "nmtam"
#define MQTT_PASSWORD "221220"
#define SENSOR_DEVICE_SRC_TOPIC  "xuong/device/kitchen/status"
#define SENSOR_DEVICE_DST_TOPIC  "xuong/device/telegram/cmd"
#define DHTPIN 4
#define DHTTYPE DHT22

DHT dht(DHTPIN, DHTTYPE);  
WiFiClient wifiClient;
PubSubClient client(wifiClient);

void setup_wifi() {
  WiFi.mode(WIFI_STA);
  WiFi.begin(ssid, password);
  Serial.print("Connecting to WiFi ..");
  while (WiFi.status() != WL_CONNECTED) {
    Serial.print('.');
    delay(1000);
  }
  Serial.println(WiFi.localIP());
}

void connectToBroker() {
  while (!client.connected()) {
    Serial.print("Attempting MQTT connection...");
    String clientId = "ESP32";
    clientId += String(random(0xffff), HEX);
    if (client.connect(clientId.c_str(), MQTT_USER, MQTT_PASSWORD)) {
      Serial.println("connected");
      client.subscribe(SENSOR_DEVICE_DST_TOPIC);
    } else {
      Serial.print("failed, rc=");
      Serial.print(client.state());
      Serial.println(" try again in 2 seconds");
      delay(2000);
    }
  }
}
 
void callback(char* topic, byte *payload, unsigned int length) {  
  Serial.println("-------New topicMsg from broker-----");
  Serial.print("topic: ");
  Serial.println(topic);
  Serial.print("topicMsg: ");
  char statusMsg[length+1];
  memcpy(statusMsg, payload, length);
  statusMsg[length] = NULL;
  String topicMsg(statusMsg);
  Serial.println();
  Serial.println(topicMsg);
  if(String(topic) == SENSOR_DEVICE_DST_TOPIC)
  {
    float t = dht.readTemperature();
    float h = dht.readHumidity();
    if (isnan(h) || isnan(t)) 
    {
      Serial.println("ERROR: Failed to read from DHT sensor!");
    }
    else 
    {
      if(topicMsg == "KITCHEN_TEMP")
      {
        client.publish(SENSOR_DEVICE_SRC_TOPIC, (String(t)+" °C").c_str());
      }
      else if(topicMsg == "KITCHEN_HUM")
      {
        client.publish(SENSOR_DEVICE_SRC_TOPIC, (String(h)+" %").c_str());
      }    
    }
  }    
}
void setup() {
  Serial.begin(115200);
  dht.begin();
  Serial.setTimeout(500);
  setup_wifi();
  client.setServer(MQTT_SERVER, MQTT_PORT);
  client.setCallback(callback);
  connectToBroker();
  Serial.println("Start transfer");
}

void loop() {
  client.loop();
  if (!client.connected()) {
    connectToBroker();
  }
}
