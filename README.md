# 开源项目地址: 
项目地址: [https://github.com/ning1875/prome-route](https://github.com/ning1875/prome-route)
PS: 这是一个仅用时半天就写完的项目
架构图 
![image](https://github.com/ning1875/prome-route/blob/master/images/prome-route.jpg)
# prometheus HA
**prometheus本地tsdb性能出色，但是碍于其没有集群版本导致**
## 实现手段
**注意这些手段都是要数据的统一存储**
- 可以通过remote_write 到一个提供HA的tsdb存储中
- 通过联邦收集到一个prometheus里

## 问题来了，搞不定集中式的tsdb集群，或者集群挂了咋办



# 本项目介绍
## 原理介绍
- 肯定有一组prometheus 服务器和pod用来采集各式各样的数据
- 那么采集器上本地的数据就是一个个分片，采集器本身也可以充当查询的角色
- 而且每个采集器上面的指标通过一个`特征标签`比如cluster/app等区分
- 通常是定义global.external_labels中的
    ```yaml
    global:
      external_labels:
        cluster: a
    ```
- 如果能有一个路由组件，知道所有特征标签对应的采集器地址
```python
shard_addr_map = {  
 "cluster_a": "1.1.1.1:9090",  
 "cluster_b": "2.2.2.2:9090",  
 "cluster_c": "3.3.3.3:9090",  
}
```
- 然后根据请求中的expr获取到`特征标签`，将其替换掉
- 因为在采集器本地存储的时候没有`特征标签`
- 转发到指定的采集器请求数据后再返回给grafana即可


## 需要适配的接口
### prometheus `3`大查询接口
- instance_query  : /api/v1/query  报警使用和当前点查询
- range_query  : /api/v1/query_range 查询一段时间的曲线
- series  ： /api/v1/series  使用label_values查询变量
对应在代码中实现
```golang
func Routes(r *gin.Engine) {  
  
   qApi := r.Group("/")  
   qApi.GET("/api/v1/query_range", promeRangequery)  
   qApi.GET("/api/v1/query", promeInstancequery)  
   qApi.GET("/api/v1/series", promeSeriesQuery)  
  
}
```
### 查询状态码不同时返回数据结构不同
**这个很好解决，用interface即可**
```golang
respBytes, err := ioutil.ReadAll(resp.Body)  
if err != nil {  
   log.Error(err.Error())  
   c.String(http.StatusInternalServerError, fmt.Sprintf(`target prome %s error by %s=%s `, targetProme, keyName, labelName))  
   return  
}  
var respInterface interface{}  
_ = json.Unmarshal(respBytes, &respInterface)  
  
c.JSON(resp.StatusCode, respInterface)
```

## 优缺点
**优点**

- 查询在各自采集器完成，不用受限于集中tsdb的可用性，挂了，也可以查到，查询互相不受影响
- 数据保存时间不受限于统一的tsdb配置，可以各自配置业务采集器
- 查询limit参数也不再首先于统一的tsdb配置，放飞。。。
- 组件无状态，只做转发，可横向扩容

**缺点**

- 受限于统一的label特征
# 使用指南
## 适用范围
- 不想维护tsdb集群
- 给tsdb集群挂了时做备份查询
- 查询时含有`特征标签`，采集器上数据没有`特征标签`
**注意**
- 如果本身每个采集器上面的数据已经有`特征标签`区别好了，那么需要改下本项目的代码直接转发即可

## 
```c  
# build  
git clone https://github.com/ning1875/prome-route.git  
go build -o prome-route main.go   
  
#修改配置文件  
补充prome-route.yml中的信息:  
replace_label_name: cluster # 特征标签，即grafana查询时用来区分不同shard的label name  
 # 比如特征标签为cluster ：node_memory_MemFree_bytes{cluster="a",node=~".+"}  
 # 代表查询分片a上的node_memory_MemFree_bytes数据  
prome_servers:  
  
 a: 1.1.1.1:9090 # 各个分片采集器的value及其地址  
 b: 2.2.2.2:9090  
  
  
http:  
 listen_addr: :9097  
 
#启动服务  
./prome-route --config.file=prome-route.yml
systemctl start prome-route

# 在grafana中添加数据源地址填 $prome-route:9097 如1.1.1.1:9097
```




