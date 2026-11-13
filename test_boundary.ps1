$base = "http://127.0.0.1:8080"
function Token($u,$p){ (Invoke-RestMethod -Uri "$base/api/login" -Method Post -Body (ConvertTo-Json @{username=[string]$u;password=[string]$p}) -ContentType "application/json").token }
function Post($h,$u,$b){ try{ Invoke-RestMethod -Uri $u -Method Post -Headers $h -Body (ConvertTo-Json $b) -ContentType "application/json" } catch { $_.ErrorDetails.Message } }
$app = Token applicant 123456; $apr = Token approver 123456; $op = Token operator 123456
$hApp=@{Authorization="Bearer $app"}; $hApr=@{Authorization="Bearer $apr"}; $hOp=@{Authorization="Bearer $op"}
$allV = @("V1","V2","V3","V4","V5")
Invoke-RestMethod -Uri "$base/api/dev/reset" -Method Post -Headers $hApp | Out-Null

echo "--- 边界：newInterval=9 ---"
$r1 = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=9;targetVehicles=$allV}
echo "resp: $r1"
echo "--- 边界：newInterval=301 ---"
$r2 = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=301;targetVehicles=$allV}
echo "resp: $r2"
echo "--- 边界：reason 为空 ---"
$r3 = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="";newInterval=30;targetVehicles=$allV}
echo "resp: $r3"
echo "--- 边界：部分选择车辆(仅V1) ---"
$r4 = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=30;targetVehicles=@("V1")}
echo "resp: $r4"

echo "--- 幂等：重复试点/推广 ---"
$c = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=30;targetVehicles=$allV}
Post $hApp "$base/api/applications/$($c.id)/submit" @{} | Out-Null
$act = (Post $hApr "$base/api/applications/$($c.id)/approve" @{}).id
Post $hOp "$base/api/activities/$act/pilot" @{vehicleIds=@("V1","V2")} | Out-Null
$st1 = (Invoke-RestMethod -Uri "$base/api/activities/$act" -Headers $hApp).activity
$p2 = Post $hOp "$base/api/activities/$act/pilot" @{vehicleIds=@("V1","V2")}
$st2 = (Invoke-RestMethod -Uri "$base/api/activities/$act" -Headers $hApp).activity
echo "pilot x2 -> $("$($st1.status) / $($st2.status)") (应一致 PILOTING)"
$vt1=(Invoke-RestMethod -Uri "$base/api/vehicle-agent/login" -Method Post -Body (ConvertTo-Json @{vehicleId="V1";secret="secret-V1"}) -ContentType "application/json").token
$vt2=(Invoke-RestMethod -Uri "$base/api/vehicle-agent/login" -Method Post -Body (ConvertTo-Json @{vehicleId="V2";secret="secret-V2"}) -ContentType "application/json").token
$c1=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$act" -Headers @{Authorization="Bearer $vt1"}
Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$c1.versionId;result="SUCCESS"} | Out-Null
$c2=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$act" -Headers @{Authorization="Bearer $vt2"}
Post @{Authorization="Bearer $vt2"} "$base/api/vehicle-agent/receipts" @{versionId=$c2.versionId;result="SUCCESS"} | Out-Null
$tl1=(Invoke-RestMethod -Uri "$base/api/activities/$act/timeline?vehicleId=V1" -Headers $hApp)
$n1 = if($tl1 -is [array]){$tl1.Count}else{$tl1.PSObject.Properties.Name.Count}
Post $hOp "$base/api/activities/$act/rollout" @{} | Out-Null
$r3b = Post $hOp "$base/api/activities/$act/rollout" @{}
$tl2=(Invoke-RestMethod -Uri "$base/api/activities/$act/timeline?vehicleId=V1" -Headers $hApp)
$n2 = if($tl2 -is [array]){$tl2.Count}else{$tl2.PSObject.Properties.Name.Count}
echo "rollout x2 -> 二次响应: $r3b"
echo "V1 版本条目数 试点前=$n1 推广后=$n2 (推广后应+1且二次推广不新增)"
