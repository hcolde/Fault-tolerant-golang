# fault-tolerant

> fault-tolerant是Golang实现的，基于[Snowflake](https://github.com/twitter-archive/snowflake)算法的唯一ID生成器。

1. fault-tolerant允许时间发生回拨，并使用一定的位数去累计系统时间发生回拨的次数；
2. fault-tolerant通过CAS操作，保证seq在1ms内唯一。

不足：由于允许时间发生回拨，因此可能无法通过生成的ID及机器启动时间计算该ID的生成时间。

## 算法

| sign | delta seconds | worker id | callback times | sequence |
| ---- | ------------- | --------- | -------------- | -------- |
| 1bit | 41bits        | 9bits     | 3bits          | 10bits   |

* sign
  * 固定1bit符号标识，即生成的ID为正数
* delta seconds
  * 增量时间，机器启动时间至当前时间的增量，单位：毫秒，最多可支持69年
* worker id
  * 机器id，最多可支持512次机器启动
* callback times
  * 时间回拨次数，最多可容许系统发生7次时间回拨
* sequence
  * 每毫秒下的并发序列，每秒可支持102.4万个并发

**以上参数均可根据实际项目需求自定义**

