// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package main 演示如何在多网卡环境下通过指定网卡连接到不同的OPC UA设备
//
// 场景：
// - 电脑有两个网卡：192.168.100.10 和 192.168.100.20
// - 两个网卡都连接了OPC UA设备，设备IP都是192.168.100.1
// - 需要通过指定的网卡连接到对应的设备
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func main() {
	var (
		endpoint1  = flag.String("endpoint1", "opc.tcp://192.168.100.1:4840", "第一个OPC UA服务器端点")
		endpoint2  = flag.String("endpoint2", "opc.tcp://192.168.100.1:4840", "第二个OPC UA服务器端点")
		localAddr1 = flag.String("local1", "192.168.100.10:0", "连接第一个设备时使用的本地网卡地址")
		localAddr2 = flag.String("local2", "192.168.100.20:0", "连接第二个设备时使用的本地网卡地址")
		nodeID     = flag.String("node", "i=2258", "要读取的节点ID")
	)
	flag.Parse()

	ctx := context.Background()

	// 连接到第一个设备（通过第一个网卡）
	fmt.Printf("通过网卡 %s 连接到设备 %s\n", *localAddr1, *endpoint1)
	client1, err := connectWithLocalAddr(ctx, *endpoint1, *localAddr1)
	if err != nil {
		log.Fatalf("连接第一个设备失败: %v", err)
	}
	defer client1.Close(ctx)

	// 连接到第二个设备（通过第二个网卡）
	fmt.Printf("通过网卡 %s 连接到设备 %s\n", *localAddr2, *endpoint2)
	client2, err := connectWithLocalAddr(ctx, *endpoint2, *localAddr2)
	if err != nil {
		log.Fatalf("连接第二个设备失败: %v", err)
	}
	defer client2.Close(ctx)

	// 从两个设备读取数据
	fmt.Println("\n=== 从第一个设备读取数据 ===")
	if err := readNodeValue(ctx, client1, *nodeID, "设备1"); err != nil {
		log.Printf("从设备1读取数据失败: %v", err)
	}

	fmt.Println("\n=== 从第二个设备读取数据 ===")
	if err := readNodeValue(ctx, client2, *nodeID, "设备2"); err != nil {
		log.Printf("从设备2读取数据失败: %v", err)
	}

	fmt.Println("\n连接测试完成！")
}

// connectWithLocalAddr 使用指定的本地地址连接到OPC UA服务器
func connectWithLocalAddr(ctx context.Context, endpoint, localAddr string) (*opcua.Client, error) {
	// 创建客户端配置，指定本地网卡地址
	opts := []opcua.Option{
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityModeString("None"),
		opcua.LocalAddr(localAddr), // 指定使用的本地网卡地址
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(5 * time.Second),
	}

	// 创建客户端
	c, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("创建客户端失败: %v", err)
	}

	// 建立连接
	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("连接服务器失败: %v", err)
	}

	fmt.Printf("成功连接到 %s (本地地址: %s)\n", endpoint, localAddr)
	return c, nil
}

// readNodeValue 从指定节点读取值
func readNodeValue(ctx context.Context, client *opcua.Client, nodeIDStr, deviceName string) error {
	// 解析节点ID
	nodeID, err := ua.ParseNodeID(nodeIDStr)
	if err != nil {
		return fmt.Errorf("解析节点ID失败: %v", err)
	}

	// 读取节点值
	req := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeID},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := client.Read(ctx, req)
	if err != nil {
		return fmt.Errorf("读取请求失败: %v", err)
	}

	if len(resp.Results) == 0 {
		return fmt.Errorf("没有读取结果")
	}

	result := resp.Results[0]
	if result.Status != ua.StatusOK {
		return fmt.Errorf("读取状态错误: %v", result.Status)
	}

	fmt.Printf("%s - 节点 %s 的值: %v (类型: %T)\n",
		deviceName, nodeIDStr, result.Value.Value(), result.Value.Value())
	fmt.Printf("%s - 服务器时间戳: %v\n", deviceName, result.ServerTimestamp)
	fmt.Printf("%s - 源时间戳: %v\n", deviceName, result.SourceTimestamp)

	return nil
}

// 示例：如何创建自定义的Dialer来实现更高级的网络控制
func createCustomDialer(localAddr string) opcua.Option {
	return opcua.LocalAddr(localAddr)
}

// 高级示例：同时监控多个设备
func monitorMultipleDevices(ctx context.Context) error {
	devices := []struct {
		endpoint  string
		localAddr string
		name      string
	}{
		{"opc.tcp://192.168.100.1:4840", "192.168.100.10:0", "设备A"},
		{"opc.tcp://192.168.100.1:4840", "192.168.100.20:0", "设备B"},
	}

	for _, device := range devices {
		go func(dev struct {
			endpoint  string
			localAddr string
			name      string
		}) {
			client, err := connectWithLocalAddr(ctx, dev.endpoint, dev.localAddr)
			if err != nil {
				log.Printf("连接 %s 失败: %v", dev.name, err)
				return
			}
			defer client.Close(ctx)

			// 持续监控设备状态
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// 读取服务器状态节点
					if err := readNodeValue(ctx, client, "i="+fmt.Sprintf("%d", id.Server_ServerStatus_State), dev.name); err != nil {
						log.Printf("%s 读取状态失败: %v", dev.name, err)
					}
				}
			}
		}(device)
	}

	return nil
}
