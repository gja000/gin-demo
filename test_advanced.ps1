$base = "http://127.0.0.1:8080"
$allV = @("V1","V2","V3","V4","V5")
$pilotV = @("V1","V2")
$rollV = @("V3","V4","V5")
function Token($u,$p){ (Invoke-RestMethod -Uri "$base/api/login" -Method Post -Body (ConvertTo-Json @{username=[string]$u;password=[string]$p}) -ContentType "application/json").token }
function VToken($v){ (Invoke-RestMethod -Uri "$base/api/vehicle-agent/login" -Method Post -Body (ConvertTo-Json @{vehicleId=[string]$v;secret="secret-$v"}) -ContentType "application/json").token }
function Call($m,$u,$uri,$body){ try { if($body){ return Invoke-RestMethod -Uri $uri -Method $m -Headers $u -Body (ConvertTo-Json $body) -ContentType "application/json" } else { return Invoke-RestMethod -Uri $uri -Method $m -Headers $u } } catch { $e=$_.ErrorDetails.Message; if(!$e){$e=$_.Exception.Message}; return @{_err=$e} } }
function Post($u,$uri,$body){ return Call POST $u $uri $body }
function GetJ($u,$uri){ return Call GET $u $uri $null }

$app = Token applicant 123456; $apr = Token approver 123456; $op = Token operator 123456
$hApp=@{Authorization="Bearer $app"}; $hApr=@{Authorization="Bearer $apr"}; $hOp=@{Authorization="Bearer $op"}

# ===== AC-03 超时 =====
echo "=== RESET (timeout) ==="
Call POST $hApp "$base/api/dev/reset" $null | Out-Null
echo "--- AC-03 timeout ---"
$c=Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=30;targetVehicles=$allV}
$aid=$c.id
Post $hApp "$base/api/applications/$aid/submit" $null | Out-Null
$actId2=(Post $hApr "$base/api/applications/$aid/approve" $null).id
Post $hOp "$base/api/activities/$actId2/pilot" @{vehicleIds=$pilotV} | Out-Null
echo "waiting 17s for timeout (TIMEOUT_SECONDS=15)..."
Start-Sleep -Seconds 17
$st=(GetJ $hApp "$base/api/activities/$actId2").activity
echo "activity status: $($st.status)"
$states=GetJ $hApp "$base/api/activities/$actId2/vehicles"
$states | ForEach-Object { if($_.batch -eq "PILOT"){ echo "VEH $($_.vehicleId): status=$($_.status) timedOut=$($_.timedOut)" } }
$rl=Post $hOp "$base/api/activities/$actId2/rollout" $null
echo "rollout -> code=$($rl.code)"

# ===== AC-08/09 恢复 + AC-11 旧回执 =====
echo "=== RESET (restore) ==="
Call POST $hApp "$base/api/dev/reset" $null | Out-Null
echo "--- build normal flow to PILOT_PASSED ---"
$c=Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=30;targetVehicles=$allV}
$aid=$c.id
Post $hApp "$base/api/applications/$aid/submit" $null | Out-Null
$actId=(Post $hApr "$base/api/applications/$aid/approve" $null).id
Post $hOp "$base/api/activities/$actId/pilot" @{vehicleIds=$pilotV} | Out-Null
$vt1=VToken V1; $vt2=VToken V2
$c1=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId" -Headers @{Authorization="Bearer $vt1"}
$c2=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId" -Headers @{Authorization="Bearer $vt2"}
Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$c1.versionId;result="SUCCESS"} | Out-Null
Post @{Authorization="Bearer $vt2"} "$base/api/vehicle-agent/receipts" @{versionId=$c2.versionId;result="SUCCESS"} | Out-Null
Start-Sleep 1
$st=(GetJ $hApp "$base/api/activities/$actId").activity
echo "after pilot both success: $($st.status)"
# 停止（V1/V2 已生效 30；前值 60）
$sp=Post $hApr "$base/api/activities/$actId/stop" @{reason="stop"}
echo "stop status=$($sp.status)"
# 恢复流程
Post $hOp "$base/api/activities/$actId/restore" $null | Out-Null
Post $hApr "$base/api/activities/$actId/restore/approve" $null | Out-Null
$ex=Post $hOp "$base/api/activities/$actId/restore/execute" $null
echo "restore execute status=$($ex.status)"
$tl=GetJ $hApp "$base/api/activities/$actId/timeline"
$tl | ForEach-Object { echo "TIMELINE veh=$($_.vehicle_id) ver=$($_.version_no) expected=$($_.expected_interval) batch=$($_.batch) status=$($_.status)" }
# V1 离线；V2 恢复成功
$vt2=VToken V2
$c2r=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId" -Headers @{Authorization="Bearer $vt2"}
Post @{Authorization="Bearer $vt2"} "$base/api/vehicle-agent/receipts" @{versionId=$c2r.versionId;result="SUCCESS"} | Out-Null
Start-Sleep 1
$cr=Post $hApr "$base/api/activities/$actId/restore/complete" $null
echo "restore complete (V1 offline) -> code=$($cr.code)"
$st=(GetJ $hApp "$base/api/activities/$actId").activity
echo "activity status now=$($st.status) fullyRestored=$($st.fullyRestored)"
# V1 上线拉取最新(恢复版本) 并成功 -> 全部确认
$vt1=VToken V1
$c1r=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId" -Headers @{Authorization="Bearer $vt1"}
echo "V1 pull after online: versionId=$($c1r.versionId) verNo=$($c1r.versionNo) expected=$($c1r.expectedInterval)"
Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$c1r.versionId;result="SUCCESS"} | Out-Null
Start-Sleep 1
$cr2=Post $hApr "$base/api/activities/$actId/restore/complete" $null
echo "restore complete (all) -> status=$($cr2.status)"
# AC-11 旧回执：对 V1 的旧 version1 上报（应忽略，生效值不变=60）
$oldReceipt=Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$c1.versionId;result="SUCCESS"}
echo "AC-11 stale receipt -> status=$($oldReceipt.status) (期望保持, 不覆盖)"
$veh=GetJ $hApp "$base/api/activities/$actId/vehicles" | Where-Object {$_.vehicleId -eq "V1"}
echo "V1 effective after stale receipt=$($veh.effectiveInterval) (应为60)"
