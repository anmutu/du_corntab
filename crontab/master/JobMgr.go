/*
  author='du'
  date='2020/5/26 9:10'
*/
package master

import (
	"context"
	"du_corntab/crontab/master/common"
	"encoding/json"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client //这里用的是指针
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	G_JobMgr *JobMgr //单例对象,指针
)

//初始化管理器
func InitJogMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)

	//初始化配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     //集群地址
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, //连接超时时间
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	//得到kv和lease租约
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	//将之赋值给单例对像
	G_JobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}

//保存job到etcd。
func (JobMgr *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)

	//拿到jobKey和jobValue,将交之保存到etcd中。
	jobKey = common.JOB_SAVE_DIR + job.Name
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	if putResp, err = JobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		//如果是更新，就返回以前的值，如果反序列化不成功，则将对象和err都置为nil。
		if putResp.PrevKv.Value != nil {
			if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
				err = nil
				oldJob = nil
			} else {
				oldJob = &oldJobObj
			}
		}
		return nil, err
	}
	return
}

//从etcd里删除job
func (JobMgr *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)

	//得到key并删除它
	jobKey = common.JOB_SAVE_DIR + name
	if delResp, err = JobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	//拿到被删除信息，返回。
	if len(delResp.PrevKvs) != 0 {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

//获取etcd里的任务
func (JobMgr *JobMgr) ListJobs() (jobList []*common.Job, err error) {
	var (
		dirKey  string
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		job     *common.Job
	)
	dirKey = common.JOB_SAVE_DIR
	if getResp, err = JobMgr.kv.Get(context.TODO(), dirKey, clientv3.WithPrevKV()); err != nil {
		return
	}

	jobList = make([]*common.Job, 0)

	//遍历所有任务
	for kvPair = range getResp.Kvs {
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			continue
		}
		jobList = append(jobList, job)
	}
	return

}