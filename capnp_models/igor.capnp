using Go = import "/go.capnp";
@0xaeb14304b9528324;
$Go.package("igor");
$Go.import("github.com/alittlebrighter/igor");

struct Event {
    timestamp @0 :UInt64; # time in milliseconds
    type @1 :EventType;
    location @2 :List(Text);
    from @3 :Text;
    payload @4 :List(KeyValue);
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
}

enum EventType {
    sensor @0;
    settingChange @1;
    stateChange @2;
    request @3;
    response @4;
}
