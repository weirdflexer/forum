// Run only against a dedicated load-test environment populated with load-fixture.sql.
// k6 run -e BASE_URL=http://localhost:18081 -e TOPIC_ID=<open-topic-uuid> tests/load.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
const readTime = new Trend('forum_read_ms', true), writeTime = new Trend('forum_write_ms', true), serverErrors = new Rate('forum_server_errors');
export const options = {
 scenarios:{traffic:{executor:'constant-arrival-rate',rate:20,timeUnit:'1s',duration:'15m',preAllocatedVUs:100,maxVUs:100}},
 thresholds:{forum_read_ms:['p(95)<500'],forum_write_ms:['p(95)<1000'],forum_server_errors:['rate<0.01'],checks:['rate>0.99'],dropped_iterations:['count==0']},
};
const base=__ENV.BASE_URL||'http://localhost:18081';let csrf='';
export default function(){
 const slot=(__ITER+__VU)%10;let res;
 if(slot===9){
  if(!csrf){const s=http.post(base+'/api/v1/sessions',null,{headers:{Origin:base}});check(s,{'session ready':r=>r.status===200});if(s.status!==200){sleep(5);return}csrf=s.json('csrf_token');}
  if(!__ENV.TOPIC_ID)throw new Error('Set TOPIC_ID to an open topic in the dedicated load database');
  res=http.post(base+'/api/v1/topics/'+__ENV.TOPIC_ID+'/posts',JSON.stringify({body:'Нагрузочная проверка '+__VU+' / '+__ITER}),{headers:{Origin:base,'Content-Type':'application/json','X-CSRF-Token':csrf,'Idempotency-Key':`load-${__VU}-${__ITER}-${Date.now()}`}});
  writeTime.add(res.timings.duration);check(res,{'write accepted':r=>r.status===201});
 }else{
  res=http.get(base+'/api/v1/topics'+(slot===8?'?q='+encodeURIComponent('нагрузочная'):''));
  readTime.add(res.timings.duration);check(res,{'read accepted':r=>r.status===200});
 }
 serverErrors.add(res.status>=500);sleep(5);
}
