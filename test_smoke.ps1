$base = "http://127.0.0.1:8080"
$allV = @("V1","V2","V3","V4","V5")
$pilotV = @("V1","V2")
$rollV = @("V3","V4","V5")

function Token($u,$p){ (Invoke-RestMethod -Uri "$base/api/login" -Method Post -Body (ConvertTo-Json @{username=[string]$u;password=[string]$p}) -ContentType "application/json").token }
function VToken($v){ (Invoke-RestMethod -Uri "$base/api/vehicle-agent/login" -Method Post -Body (ConvertTo-Json @{vehicleId=[string]$v;secret="secret-$v"}) -ContentType "application/json").token }

$app = Token applicant 123456
$apr = Token approver 123456
$op  = Token operator 123456
$hApp = @{Authorization="Bearer $app"}
$hApr = @{Authorization="Bearer $apr"}
$hOp  = @{Authorization="Bearer $op"}

# 重置开发数据
Invoke-RestMethod -Uri "$base/api/dev/reset" -Method Post -Headers $hApp | Out-Null
echo "RESET done"

$body = @{fleetId="FLEET-A"; reason="check AEB"; newInterval=30; targetVehicles=$allV}
$create = Invoke-RestMethod -Uri "$base/api/applications" -Method Post -Headers $hApp -Body (ConvertTo-Json $body) -ContentType "application/json"
echo "CREATE: $($create.id) status=$($create.status)"
$appId = $create.id

$sub = Invoke-RestMethod -Uri "$base/api/applications/$appId/submit" -Method Post -Headers $hApp
echo "SUBMIT: status=$($sub.status) submitCount=$($sub.submitCount)"

$act = Invoke-RestMethod -Uri "$base/api/applications/$appId/approve" -Method Post -Headers $hApr
echo "APPROVE: act=$($act.id) status=$($act.status) newInterval=$($act.newInterval)"
$actId = $act.id

$pilot = Invoke-RestMethod -Uri "$base/api/activities/$actId/pilot" -Method Post -Headers $hOp -Body (ConvertTo-Json @{vehicleIds=$pilotV}) -ContentType "application/json"
echo "PILOT: status=$($pilot.status)"

foreach($v in $pilotV){
  $vt = VToken $v
  $cfg = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId" -Headers @{Authorization="Bearer $vt"}
  echo "PULL $v : hasConfig=$($cfg.hasConfig) versionId=$($cfg.versionId) expected=$($cfg.expectedInterval)"
  if($cfg.hasConfig){
    $rc = Invoke-RestMethod -Uri "$base/api/vehicle-agent/receipts" -Method Post -Headers @{Authorization="Bearer $vt"} -Body (ConvertTo-Json @{versionId=$cfg.versionId;result="SUCCESS"}) -ContentType "application/json"
    echo "RECEIPT $v : status=$($rc.status) dup=$($rc.duplicate)"
  }
}

$act2 = (Invoke-RestMethod -Uri "$base/api/activities/$actId" -Headers $hApp).activity
echo "AFTER PILOT: status=$($act2.status)"

$roll = Invoke-RestMethod -Uri "$base/api/activities/$actId/rollout" -Method Post -Headers $hOp
echo "ROLLOUT: status=$($roll.status)"

foreach($v in $rollV){
  $vt = VToken $v
  $cfg = Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId" -Headers @{Authorization="Bearer $vt"}
  if($cfg.hasConfig){
    $rc = Invoke-RestMethod -Uri "$base/api/vehicle-agent/receipts" -Method Post -Headers @{Authorization="Bearer $vt"} -Body (ConvertTo-Json @{versionId=$cfg.versionId;result="SUCCESS"}) -ContentType "application/json"
    echo "RECEIPT $v : status=$($rc.status)"
  } else { echo "PULL $v : NO CONFIG" }
}

$comp = Invoke-RestMethod -Uri "$base/api/activities/$actId/complete" -Method Post -Headers $hApr
echo "COMPLETE: status=$($comp.status)"

$ov = Invoke-RestMethod -Uri "$base/api/activities/$actId/overview" -Headers $hApp
echo "OVERVIEW: $(ConvertTo-Json $ov -Compress)"
$states = Invoke-RestMethod -Uri "$base/api/activities/$actId/vehicles" -Headers $hApp
$states | ForEach-Object { echo "VEH $($_.vehicleId): status=$($_.status) eff=$($_.effectiveInterval) prev=$($_.prevInterval) ver=$($_.versionNo)" }
