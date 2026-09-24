fn shade(uv: vec2<f32>) -> vec3<f32> {
 let t=uni.time*uni.speed; let p=uv*uni.scale;
 let v=sin(p.x*3.0+t)+sin(p.y*4.0-t)+sin(length(p)*3.0+t*uni.detail);
 return palette(v*0.18);
}
