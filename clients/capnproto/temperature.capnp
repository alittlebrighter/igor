@0xfa9c162ef5f3c740;

using Go = import "/go.capnp";
$Go.package("models");
$Go.import("github.com/alittlebrighter/igor/models");

struct Temperature {
    degrees @0 :Float64;
    unit @1 :TemperatureUnit;
}

enum TemperatureUnit {
    celsius @0;
    fahrenheight @1;
}