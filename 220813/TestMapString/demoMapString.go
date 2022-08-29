package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"    
    // "github.com/fatih/structs"
)


var mqttClientHandleTele mqtt.Client
var mqttClientHandleSerial mqtt.Client

var  serialRXChannel chan string

type Mqtt struct {
    Broker string

    SerialSrcTopic string
    SerialDstTopic string

    TeleSrcTopic string  
    TeleDstTopic string   
}


type LedControlCode struct {
    TokenCode []string
    Cmd string
    ResponseMap map[string]string
}

type Command struct {
    ControlLed []LedControlCode

    DefaultRespMsg string
    ConnectionLostResMsg string
    Timeout time.Duration
}

type FileConfig struct {
    MqttConfig Mqtt
    //CmdConfig []LedControlCode
    CmdConfig Command
}

var cfg FileConfig

func yamlFileHandle() {
    yfile, err := ioutil.ReadFile("config.yaml")

    if err != nil {

      log.Fatal(err)
    }

    err2 := yaml.Unmarshal(yfile, &cfg)

    if err2 != nil {
        fmt.Println("Error file yaml")

      log.Fatal(err2)
    }
    // fmt.Println(cfg)
} 

func handleTeleCmd(msg string) {

    // switch msg {
    //     case cfg.CmdConfig.Led1OnMsgVN.TokenCode[0], 
    //          cfg.CmdConfig.Led1OnMsgVN.TokenCode[1]:

    //         handleTeleScript(cfg.CmdConfig.Led1OnMsgVN)

    //     case cfg.CmdConfig.Led1OnMsgEN.TokenCode[0], 
    //          cfg.CmdConfig.Led1OnMsgEN.TokenCode[1]:
            
    //         handleTeleScript(cfg.CmdConfig.Led1OnMsgEN)
}



func main() {

    yamlFileHandle()

    // cmdList := map[string]LedControlCode{}

    // for _, script := range cfg.CmdConfig.ControlLed {
    //     //fmt.Println(key, value)
    //     for _, msgTele := range script.TokenCode {
    //         //fmt.Println(msgTele)
    //         cmdList[msgTele] = script
    //         fmt.Println(cmdList[msgTele].TokenCode)
    //         fmt.Println("===========")
    //     } 
    //     //fmt.Println(script.TokenCode[0])

    // }


    // cmdList := map[string]LedControlCode{}

    // var cmdConfigMap = map[string]map[string]LedControlCode{}

    // names := structs.Names(&Command{})

    // for _, name := range names {
    //     fmt.Printf("name = %s\n", name)
    // }

    // m := structs.Map(&cfg.CmdConfig)
    // //fmt.Println(m)
    // // for _, value := range m {
    // //     //fmt.Println(key, value)
    //     fmt.Println(m["Led1OnMsgVN"].(map[string]interface{})["TokenCode"])
    // //}

    // fmt.Println(cfg.CmdConfig.Led1OnMsgVN.TokenCode[0])



    // for _, value := range cfg.CmdConfig.Led1OnMsgVN.TokenCode {
    //     //fmt.Println(key, value)
    //     cmdList[value] = cfg.CmdConfig.Led1OnMsgVN
    // }

    // // cmdConfigMap["Led1OnMsgVN"] = 

    // for _, value := range cfg.CmdConfig.Led1OnMsgEN.TokenCode {
    //     //fmt.Println(key, value)
    //     cmdList[value] = cfg.CmdConfig.Led1OnMsgEN
    // }

    // // cmdList["Bật đèn 1"] = cfg.CmdConfig.Led1OnMsgVN
    // // cmdList["Bat den 1"] = cfg.CmdConfig.Led1OnMsgVN

    // fmt.Println(cmdList)

}