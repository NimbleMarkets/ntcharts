fn shade(uv: vec2<f32>) -> vec3<f32> {
 let t=uni.time*uni.speed; let p=rotate(uv,t*0.2)*uni.scale;
 let a=atan2(p.y,p.x); let r=length(p);
 let v=sin(cos(a*8.0)*r*3.0+uni.detail*sin(r*4.0-t))+cos(r*5.0-t);
 let lines=pow(0.5+0.5*cos(v*4.0),8.0);
 return palette(v*0.25+t*0.07)*(0.55+0.45*lines)+vec3<f32>(lines*0.2);
}
