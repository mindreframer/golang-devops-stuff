package yagnats

type Logger interface {
	Fatal(string)
	Error(string)
	Warn(string)
	Info(string)
	Debug(string)

	Fatald(map[string]interface{}, string)
	Errord(map[string]interface{}, string)
	Warnd(map[string]interface{}, string)
	Infod(map[string]interface{}, string)
	Debugd(map[string]interface{}, string)
}

type DefaultLogger struct{}

func (dl *DefaultLogger) Fatal(string) {}
func (dl *DefaultLogger) Error(string) {}
func (dl *DefaultLogger) Warn(string)  {}
func (dl *DefaultLogger) Info(string)  {}
func (dl *DefaultLogger) Debug(string) {}

func (dl *DefaultLogger) Fatald(map[string]interface{}, string) {}
func (dl *DefaultLogger) Errord(map[string]interface{}, string) {}
func (dl *DefaultLogger) Warnd(map[string]interface{}, string)  {}
func (dl *DefaultLogger) Infod(map[string]interface{}, string)  {}
func (dl *DefaultLogger) Debugd(map[string]interface{}, string) {}
