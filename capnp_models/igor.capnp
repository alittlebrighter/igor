@0xaeb14304b9528324;

using Go = import "/go.capnp";
$Go.package("igor");
$Go.import("github.com/alittlebrighter/igor");

struct KeyValue {
    key @0 :Text;
    value :union {
        bool @1 :Bool;
        int @2 :Int32;
        uint @3 :UInt32;
        float @4 :Float32;
        string @5 :Text;
    }
}

struct Address {
    host @0 :Text;
    sensor @1 :Text;
}

struct EventMeta {
    timestamp @0 :UInt64; # time in milliseconds
    type @1 :EventType;
    location @2 :List(Text);
    source @3 :Address;
}

struct SensorEvent {
    meta @0 :EventMeta;
    payload @1 :List(KeyValue);
}

enum EventType {
    sensorUpdate @0;
    command @1;
    response @2;
    hearbeat @3;
}
