fn shade(uv: vec2<f32>) -> vec3<f32> {
 let t=uni.time*uni.speed;
 var z=rotate(uv,t*0.05)*1.35/uni.scale;
 let c=vec2<f32>(-0.745+0.035*sin(t*0.3),0.186+0.025*cos(t*0.27));
 var count=0.0; var trap=10.0;
 for(var i=0;i<100;i=i+1) {
  z=vec2<f32>(z.x*z.x-z.y*z.y,2.0*z.x*z.y)+c;
  trap=min(trap,abs(length(z)-0.5));
  count=f32(i);
  if(dot(z,z)>256.0) { break; }
 }
 if(count>98.0) { return palette(trap*4.0+t*0.025)*exp(-trap*8.0)*0.3; }
 let smooth=count+1.0-log2(max(log2(max(length(z),1.001)),0.001));
 return palette(smooth*(0.025+uni.detail*0.018)+t*0.03)*(0.45+0.55*exp(-trap*3.0));
}
