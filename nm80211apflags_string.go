// Code generated by "stringer -type=Nm80211APFlags"; DO NOT EDIT

package gonetworkmanager

import "fmt"

const _Nm80211APFlags_name = "Nm80211APFlagsNoneNm80211APFlagsPrivacy"

var _Nm80211APFlags_index = [...]uint8{0, 18, 39}

func (i Nm80211APFlags) String() string {
	if i >= Nm80211APFlags(len(_Nm80211APFlags_index)-1) {
		return fmt.Sprintf("Nm80211APFlags(%d)", i)
	}
	return _Nm80211APFlags_name[_Nm80211APFlags_index[i]:_Nm80211APFlags_index[i+1]]
}
