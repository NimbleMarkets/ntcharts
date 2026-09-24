struct Uniforms {
 time: f32, speed: f32, scale: f32, color: f32,
 width: u32, height: u32, detail: f32, pad: f32,
}
@group(0) @binding(0) var<uniform> uni: Uniforms;
@group(0) @binding(1) var<storage, read_write> pixels: array<u32>;
fn palette(t: f32) -> vec3<f32> {
 return 0.5+0.5*cos(6.2831853*(vec3<f32>(t+uni.color)+vec3<f32>(0.0,0.33,0.67)));
}
fn rotate(p: vec2<f32>, a: f32) -> vec2<f32> {
 let c=cos(a); let s=sin(a); return vec2<f32>(c*p.x-s*p.y,s*p.x+c*p.y);
}
fn hash(p: vec2<f32>) -> f32 { return fract(sin(dot(p,vec2<f32>(127.1,311.7)))*43758.5453); }
fn noise(p: vec2<f32>) -> f32 {
 let i=floor(p); let f=fract(p); let u=f*f*(3.0-2.0*f);
 return mix(mix(hash(i),hash(i+vec2<f32>(1.0,0.0)),u.x),mix(hash(i+vec2<f32>(0.0,1.0)),hash(i+vec2<f32>(1.0)),u.x),u.y);
}
fn fbm(p: vec2<f32>) -> f32 {
 var q=p; var value=0.0; var amplitude=0.5;
 for(var i=0;i<5;i=i+1) { value=value+amplitude*noise(q); q=rotate(q,0.5)*2.03+vec2<f32>(7.1,3.8); amplitude=amplitude*0.5; }
 return value;
}
// SHADER_BODY
@compute @workgroup_size(8,8,1)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
 if(gid.x>=uni.width || gid.y>=uni.height) { return; }
 let uv=(vec2<f32>(gid.xy)+vec2<f32>(0.5)-vec2<f32>(f32(uni.width),f32(uni.height))*0.5)/f32(uni.height)*2.0;
 let rgb=vec3<u32>(clamp(shade(uv),vec3<f32>(0.0),vec3<f32>(1.0))*255.0);
 pixels[gid.y*uni.width+gid.x]=rgb.x|(rgb.y<<8u)|(rgb.z<<16u)|(255u<<24u);
}
