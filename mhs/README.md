# MHS (Mock HEC Sink)

MHS provides a simple API covering some of HEC endpoints and authentication methods. It is intended to be used for testing and development purposes.

> Splunk and Splunk HTTP Event Collector (HEC) are trademarks of Splunk Inc.

Supported Endpoints:
- [x] /services/collector
- [x] /services/collector/event
- [ ] /services/collector/event/1.0
- [ ] /services/collector/event/1.0/raw
- [ ] /services/collector/event/1.0/validate
- [ ] /services/collector/event/1.0/validate_bulk
- [ ] /services/collector/health/1.0

Supported Authentication Methods:
- [x] Token-based (HTTP Authorization header)
- [ ] Token-based (Query parameter)
- [ ] Basic Authentication (x:TOKEN)


https://docs.splunk.com/Documentation/Splunk/9.3.2/Data/HECExamples


