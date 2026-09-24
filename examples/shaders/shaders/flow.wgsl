fn shade(uv: vec2<f32>) -> vec3<f32> {
 let t=uni.time*uni.speed; let p=uv*uni.scale;
 let q=vec2<f32>(fbm(p+vec2<f32>(t*0.2,0.0)),fbm(p+vec2<f32>(4.7,-t*0.15)));
 let v=fbm(p+q*uni.detail*4.0+vec2<f32>(t*0.1));
 return palette(v*2.0)*(0.45+v)+vec3<f32>(pow(v,6.0)*0.7);
}
