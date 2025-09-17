package httpx

func SafeErrMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
