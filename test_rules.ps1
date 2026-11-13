$base = "http://127.0.0.1:8080"
$allV = @("V1","V2","V3","V4","V5")
$pilotV = @("V1","V2")
function Token($u,$p){ (Invoke-RestMethod -Uri "$base/api/login" -Method Post -Body (ConvertTo-Json @{username=[string]$u;password=[string]$p}) -ContentType "application/json").token }
function VToken($v){ (Invoke-RestMethod -Uri "$base/api/vehicle-agent/login" -Method Post -Body (ConvertTo-Json @{vehicleId=[string]$v;secret="secret-$v"}) -ContentType "application/json").token }
function Call($m,$u,$uri,$body){ try { if($body){ return Invoke-RestMethod -Uri $uri -Method $m -Headers $u -Body (ConvertTo-Json $body) -ContentType "application/json" } else { return Invoke-RestMethod -Uri $uri -Method $m -Headers $u } } catch { $e=$_.ErrorDetails.Message; if(!$e){$e=$_.Exception.Message}; return @{_err=$e} } }
function Post($u,$uri,$body){ return Call POST $u $uri $body }
function Patch($u,$uri,$body){ return Call PATCH $u $uri $body }
function GetJ($u,$uri){ return Call GET $u $uri $null }

$app = Token applicant 123456
$apr = Token approver 123456
$op  = Token operator 123456
$hApp=@{Authorization="Bearer $app"}; $hApr=@{Authorization="Bearer $apr"}; $hOp=@{Authorization="Bearer $op"}

# ===== Group1: AC-04 reject+resubmit + AC-06 param freeze =====
echo "=== RESET (group1) ==="
Call POST $hApp "$base/api/dev/reset" $null | Out-Null
echo "--- AC-04 reject + resubmit ---"
$c=Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r1";newInterval=30;targetVehicles=$allV}
$aid=$c.id
Post $hApp "$base/api/applications/$aid/submit" $null | Out-Null
Post $hApr "$base/api/applications/$aid/reject" @{reason="need scope"} | Out-Null
Patch $hApp "$base/api/applications/$aid" @{reason="r2-updated";newInterval=45} | Out-Null
Post $hApp "$base/api/applications/$aid/submit" $null | Out-Null
$g=GetJ $hApp "$base/api/applications/$aid"
echo "submitCount=$($g.application.submitCount) snapshots=$($g.snapshots.Count) newInterval=$($g.application.newInterval)"
echo "--- AC-06 param frozen ---"
Post $hApr "$base/api/applications/$aid/approve" $null | Out-Null
$f=Patch $hApp "$base/api/applications/$aid" @{newInterval=99}
echo "after approve, PATCH -> code=$($f.code) err=$($f._err)"

# ===== Group2: AC-13/12/02/07 =====
echo "=== RESET (group2) ==="
Call POST $hApp "$base/api/dev/reset" $null | Out-Null
echo "--- AC-13 cross-vehicle receipt rejected ---"
$c2=Post $hApp "$base/api/applications" @{fleetId="FLEET-A";reason="r";newInterval=30;targetVehicles=$allV}
$aid2=$c2.id
Post $hApp "$base/api/applications/$aid2/submit" $null | Out-Null
$actId2=(Post $hApr "$base/api/applications/$aid2/approve" $null).id
Post $hOp "$base/api/activities/$actId2/pilot" @{vehicleIds=$pilotV} | Out-Null
$vt1=VToken V1; $vt2=VToken V2
$cfg1=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId2" -Headers @{Authorization="Bearer $vt1"}
$cfg2=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId2" -Headers @{Authorization="Bearer $vt2"}
$bad=Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$cfg2.versionId;result="SUCCESS"}
echo "cross receipt -> code=$($bad.code)"
$aud=GetJ $hApp "$base/api/activities/$actId2/audit" | Where-Object {$_.action -eq "RECEIPT_REJECTED"}
echo "reject audit entries: $($aud.Count)"
echo "--- AC-12 duplicate receipt idempotent ---"
$g1=Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$cfg1.versionId;result="SUCCESS"}
$g2=Post @{Authorization="Bearer $vt1"} "$base/api/vehicle-agent/receipts" @{versionId=$cfg1.versionId;result="SUCCESS"}
echo "1st status=$($g1.status) ; 2nd status=$($g2.status) dup=$($g2.duplicate) dupCount=$($g2.duplicateCount)"
echo "--- AC-02 pilot fail blocks rollout ---"
$cf2=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId2" -Headers @{Authorization="Bearer $vt2"}
Post @{Authorization="Bearer $vt2"} "$base/api/vehicle-agent/receipts" @{versionId=$cf2.versionId;result="FAILED";failReason="APPLY_ERROR"} | Out-Null
Start-Sleep -Seconds 1
$st=(GetJ $hApp "$base/api/activities/$actId2").activity
echo "activity status after pilot fail: $($st.status)"
$rl=Post $hOp "$base/api/activities/$actId2/rollout" $null
echo "rollout -> code=$($rl.code)"
$cfg3=Invoke-RestMethod -Uri "$base/api/vehicle-agent/config?activityId=$actId2" -Headers @{Authorization="Bearer $(VToken V3)"}
echo "V3 pull (not in batch) -> hasConfig=$($cfg3.hasConfig) code=$($cfg3.code)"
echo "--- AC-07 stop semantics ---"
$sp=Post $hApr "$base/api/activities/$actId2/stop" @{reason="stop test"}
echo "stop status=$($sp.status)"
$states=GetJ $hApp "$base/api/activities/$actId2/vehicles"
$states | ForEach-Object { echo "VEH $($_.vehicleId): status=$($_.status) eff=$($_.effectiveInterval)" }
$ov=GetJ $hApp "$base/api/activities/$actId2/overview"
echo "overview: $(ConvertTo-Json $ov -Compress)"
