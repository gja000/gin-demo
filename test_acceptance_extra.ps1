$base = "http://127.0.0.1:8080"
function Token($u,$p){ (Invoke-RestMethod -Uri "$base/api/login" -Method Post -Body (ConvertTo-Json @{username=[string]$u;password=[string]$p}) -ContentType "application/json").token }
function VToken($v){ (Invoke-RestMethod -Uri "$base/api/vehicle-agent/login" -Method Post -Body (ConvertTo-Json @{vehicleId=[string]$v;secret="secret-$v"}) -ContentType "application/json").token }
function Post($h,$u,$b){ try{ Invoke-RestMethod -Uri $u -Method Post -Headers $h -Body (ConvertTo-Json $b) -ContentType "application/json" } catch { $_.ErrorDetails.Message } }
function GetJ($h,$u){ try{ Invoke-RestMethod -Uri $u -Headers $h } catch { $_.ErrorDetails.Message } }

$app = Token applicant 123456; $apr = Token approver 123456; $op = Token operator 123456
$hApp=@{Authorization="Bearer $app"}; $hApr=@{Authorization="Bearer $apr"}; $hOp=@{Authorization="Bearer $op"}
$allV = @("V1","V2","V3","V4","V5")

Invoke-RestMethod -Uri "$base/api/dev/reset" -Method Post -Headers $hApp | Out-Null
echo "=== RESET done ==="

function NewActivity($interval){
  $c = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=$interval;targetVehicles=$allV}
  Post $hApp "$base/api/applications/$($c.id)/submit" @{} | Out-Null
  return (Post $hApr "$base/api/applications/$($c.id)/approve" @{}).id
}

# ---------- AC-05 自审拦截 ----------
echo "--- AC-05 self-approval blocked ---"
$c5 = Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=30;targetVehicles=$allV}
Post $hApp "$base/api/applications/$($c5.id)/submit" @{} | Out-Null
$r5 = Post $hApp "$base/api/applications/$($c5.id)/approve" @{}
echo "applicant self-approve -> $r5"
$st5 = (GetJ $hApp "$base/api/applications/$($c5.id)").status
echo "application status after: $st5 (应为 SUBMITTED)"

# ---------- AC-14 超时后迟到成功回执 ----------
echo "--- AC-14 late SUCCESS after TIMEOUT ---"
$actA = NewActivity 30
Post $hOp "$base/api/activities/$actA/pilot" @{vehicleIds=@("V1","V2")} | Out-Null
echo "waiting 18s for timeout..."
Start-Sleep 18
$stA = (GetJ $hApp "$base/api/activities/$actA").activity
echo "after timeout: act=$($stA.status) (应为 PAUSED)"
$vt2 = VToken "V2"
$cfg2 = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actA" -Headers @{Authorization="Bearer $vt2"}
$rc2 = Post @{Authorization="Bearer $vt2"} "$base/api/vehicle-agent/receipts" @{versionId=$cfg2.versionId;result="SUCCESS"}
echo "late receipt -> status=$($rc2.status)"
$vehA = GetJ $hApp "$base/api/activities/$actA/vehicles"
$v2 = $vehA | Where-Object { $_.vehicleId -eq "V2" }
echo "V2 now: status=$($v2.status) eff=$($v2.effectiveInterval) timedOut=$($v2.timedOut) (应 SUCCESS/30/True 保留超时历史)"

# ---------- AC-15 停止后迟到回执 ----------
echo "--- AC-15 late receipt after STOP ---"
Invoke-RestMethod -Uri "$base/api/dev/reset" -Method Post -Headers $hApp | Out-Null
$actB = NewActivity 30
Post $hOp "$base/api/activities/$actB/pilot" @{vehicleIds=@("V1","V2")} | Out-Null
$vt1 = VToken "V1"; $vt2b = VToken "V2"
$cfg1 = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actB" -Headers @{Authorization="Bearer $vt1"}
Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$cfg1.versionId;result="SUCCESS"} | Out-Null
$rStop = Post $hApr "$base/api/activities/$actB/stop" @{reason="验收停止"}
echo "stop -> $((GetJ $hApp "$base/api/activities/$actB").activity.status)"
$cfg2b = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actB" -Headers @{Authorization="Bearer $vt2b"}
$rcB = Post @{Authorization="Bearer $vt2b"} "$base/api/vehicle-agent/receipts" @{versionId=$cfg2b.versionId;result="SUCCESS"}
echo "V2 late receipt -> status=$($rcB.status)"
$stB = (GetJ $hApp "$base/api/activities/$actB").activity
echo "activity after late receipt: $($stB.status) (应仍为 STOPPED，不自动重启/完成)"
$tl = GetJ $hApp "$base/api/activities/$actB/timeline?vehicleId=V2"
echo "V2 timeline: $($tl | ConvertTo-Json -Compress -Depth 4 | Out-String)".Trim()

# ---------- AC-17 完成门禁 ----------
echo "--- AC-17 complete gate ---"
Invoke-RestMethod -Uri "$base/api/dev/reset" -Method Post -Headers $hApp | Out-Null
$actC = NewActivity 30
Post $hOp "$base/api/activities/$actC/pilot" @{vehicleIds=@("V1","V2")} | Out-Null
$cC1 = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actC" -Headers @{Authorization="Bearer $vt1"}
Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$cC1.versionId;result="SUCCESS"} | Out-Null
$cC2 = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actC" -Headers @{Authorization="Bearer $vt2b"}
Post @{Authorization="Bearer $vt2b"} "$base/api/vehicle-agent/receipts" @{versionId=$cC2.versionId;result="FAILED";failReason="APPLY_ERROR"} | Out-Null
$rC = Post $hApr "$base/api/activities/$actC/complete" @{}
echo "complete with unconfirmed -> $rC"
$stC = (GetJ $hApp "$base/api/activities/$actC").activity
echo "activity status: $($stC.status) (应非 COMPLETED)"
