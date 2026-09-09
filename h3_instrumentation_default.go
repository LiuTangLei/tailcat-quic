//go:build !tailcat_perf

package tailcat

import "tailscale.com/wgengine/wgtransport"

func instrumentH3Factory(f wgtransport.Factory) wgtransport.Factory { return f }
